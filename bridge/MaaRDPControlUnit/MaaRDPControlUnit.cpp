// A MaaWin32ControlUnit replacement for MaaGO's RDP mode.
//
// MaaCore's Win32Controller only needs the ABI-compatible MaaFwControlUnitAPI
// vtable. RDP mode sets MAAGO_RDP_BRIDGE_ADDR and passes hwnd=0; this DLL then
// streams frames and input over a loopback TCP bridge owned by MaaGO. Local
// Win32 mode passes a real HWND and is forwarded to the original DLL, kept next
// to this one as MaaWin32ControlUnit.official.dll.

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>

#include <opencv2/core/mat.hpp>

#include <cstdint>
#include <cstdlib>
#include <cstring>
#include <mutex>
#include <string>
#include <vector>

namespace
{
constexpr uint8_t kOpConnect = 1;
constexpr uint8_t kOpConnected = 2;
constexpr uint8_t kOpUuid = 3;
constexpr uint8_t kOpFeatures = 4;
constexpr uint8_t kOpScreencap = 5;
constexpr uint8_t kOpClick = 6;
constexpr uint8_t kOpSwipe = 7;
constexpr uint8_t kOpTouchDown = 8;
constexpr uint8_t kOpTouchMove = 9;
constexpr uint8_t kOpTouchUp = 10;
constexpr uint8_t kOpKeyDown = 11;
constexpr uint8_t kOpKeyUp = 12;
constexpr uint8_t kOpClickKey = 13;
constexpr uint8_t kOpInputText = 14;
constexpr uint8_t kOpInactive = 15;
constexpr uint8_t kOpStartApp = 16;
constexpr uint8_t kOpStopApp = 17;

// Must stay byte-for-byte layout-compatible with
// MaaAssistantArknights/src/MaaCore/Controller/MaaFwControlUnitInterface.h.
class ControlUnitAPI
{
public:
    virtual ~ControlUnitAPI() = default;

    virtual bool connect() = 0;
    virtual bool connected() const = 0;

    virtual bool request_uuid(std::string& uuid) = 0;
    virtual uint64_t get_features() const = 0;

    virtual bool start_app(const std::string& intent) = 0;
    virtual bool stop_app(const std::string& intent) = 0;

    virtual bool screencap(cv::Mat& image) = 0;

    virtual bool click(int x, int y) = 0;
    virtual bool swipe(int x1, int y1, int x2, int y2, int duration) = 0;

    virtual bool touch_down(int contact, int x, int y, int pressure) = 0;
    virtual bool touch_move(int contact, int x, int y, int pressure) = 0;
    virtual bool touch_up(int contact) = 0;

    virtual bool click_key(int key) = 0;
    virtual bool input_text(const std::string& text) = 0;

    virtual bool key_down(int key) = 0;
    virtual bool key_up(int key) = 0;

    virtual bool inactive() = 0;

    virtual void* get_info() const = 0;
};

bool send_all(SOCKET sock, const void* data, size_t len)
{
    const auto* p = static_cast<const char*>(data);
    while (len > 0) {
        const int chunk = len > 1 << 20 ? 1 << 20 : static_cast<int>(len);
        const int n = ::send(sock, p, chunk, 0);
        if (n <= 0) {
            return false;
        }
        p += n;
        len -= static_cast<size_t>(n);
    }
    return true;
}

bool recv_all(SOCKET sock, void* data, size_t len)
{
    auto* p = static_cast<char*>(data);
    while (len > 0) {
        const int chunk = len > 1 << 20 ? 1 << 20 : static_cast<int>(len);
        const int n = ::recv(sock, p, chunk, 0);
        if (n <= 0) {
            return false;
        }
        p += n;
        len -= static_cast<size_t>(n);
    }
    return true;
}

bool write_u8(SOCKET sock, uint8_t value)
{
    return send_all(sock, &value, sizeof(value));
}

bool read_u8(SOCKET sock, uint8_t& value)
{
    return recv_all(sock, &value, sizeof(value));
}

bool write_u32(SOCKET sock, uint32_t value)
{
    value = htonl(value);
    return send_all(sock, &value, sizeof(value));
}

bool read_u32(SOCKET sock, uint32_t& value)
{
    if (!recv_all(sock, &value, sizeof(value))) {
        return false;
    }
    value = ntohl(value);
    return true;
}

bool write_bool(SOCKET sock, bool value)
{
    return write_u8(sock, value ? 1 : 0);
}

bool read_bool(SOCKET sock, bool& value)
{
    uint8_t raw = 0;
    if (!read_u8(sock, raw)) {
        return false;
    }
    value = raw != 0;
    return true;
}

bool write_string(SOCKET sock, const std::string& text)
{
    if (text.size() > 1 << 20) {
        return false;
    }
    return write_u32(sock, static_cast<uint32_t>(text.size())) &&
           send_all(sock, text.data(), text.size());
}

std::wstring module_path()
{
    HMODULE module = nullptr;
    if (!GetModuleHandleExW(
            GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS | GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT,
            reinterpret_cast<LPCWSTR>(&module_path),
            &module)) {
        return {};
    }
    std::wstring path(32768, L'\0');
    const DWORD len = GetModuleFileNameW(module, path.data(), static_cast<DWORD>(path.size()));
    if (len == 0) {
        return {};
    }
    path.resize(len);
    return path;
}

bool split_endpoint(const std::string& endpoint, std::string& host, std::string& port)
{
    const auto colon = endpoint.rfind(':');
    if (colon == std::string::npos) {
        return false;
    }
    host = endpoint.substr(0, colon);
    port = endpoint.substr(colon + 1);
    return !host.empty() && !port.empty();
}

class RdpUnit final : public ControlUnitAPI
{
public:
    explicit RdpUnit(std::string endpoint) : endpoint_(std::move(endpoint)) {}

    ~RdpUnit() override
    {
        std::lock_guard lock(io_);
        close_socket();
        if (wsa_started_) {
            WSACleanup();
        }
    }

    bool connect() override
    {
        std::lock_guard lock(io_);
        return ensure_connected_locked();
    }

    bool connected() const override
    {
        std::lock_guard lock(io_);
        return sock_ != INVALID_SOCKET;
    }

    bool request_uuid(std::string& uuid) override
    {
        uuid = "maago-rdp";
        return true;
    }

    uint64_t get_features() const override
    {
        // UseMouseDownAndUpInsteadOfClick | UseKeyboardDownAndUpInsteadOfClick
        return 3;
    }

    bool start_app(const std::string& intent) override
    {
        return command_string(kOpStartApp, intent);
    }

    bool stop_app(const std::string& intent) override
    {
        return command_string(kOpStopApp, intent);
    }

    bool screencap(cv::Mat& image) override
    {
        try {
            std::lock_guard lock(io_);
            if (!ensure_connected_locked() || !write_u8(sock_, kOpScreencap)) {
                return false;
            }
            bool ok = false;
            if (!read_bool(sock_, ok) || !ok) {
                return false;
            }
            uint32_t w = 0, h = 0, len = 0;
            if (!read_u32(sock_, w) || !read_u32(sock_, h) || !read_u32(sock_, len)) {
                return false;
            }
            if (w == 0 || h == 0 || len != w * h * 3 || len > 64u * 1024 * 1024) {
                return false;
            }
            // MaaCore expects the same 3-channel BGR image the official
            // screencaps return (they all end with bgra_to_bgr()).
            cv::Mat frame(static_cast<int>(h), static_cast<int>(w), CV_8UC3);
            if (frame.total() * frame.elemSize() != len) {
                return false;
            }
            if (!recv_all(sock_, frame.data, len)) {
                return false;
            }
            image = std::move(frame);
            return true;
        }
        catch (...) {
            return false;
        }
    }

    bool click(int x, int y) override
    {
        std::lock_guard lock(io_);
        return ensure_connected_locked() && write_u8(sock_, kOpClick) &&
               write_u32(sock_, static_cast<uint32_t>(x)) &&
               write_u32(sock_, static_cast<uint32_t>(y)) && read_ok_locked();
    }

    bool swipe(int x1, int y1, int x2, int y2, int duration) override
    {
        std::lock_guard lock(io_);
        return ensure_connected_locked() && write_u8(sock_, kOpSwipe) &&
               write_u32(sock_, static_cast<uint32_t>(x1)) &&
               write_u32(sock_, static_cast<uint32_t>(y1)) &&
               write_u32(sock_, static_cast<uint32_t>(x2)) &&
               write_u32(sock_, static_cast<uint32_t>(y2)) &&
               write_u32(sock_, static_cast<uint32_t>(duration)) && read_ok_locked();
    }

    bool touch_down(int contact, int x, int y, int pressure) override
    {
        (void)contact;
        (void)pressure;
        std::lock_guard lock(io_);
        return ensure_connected_locked() && write_u8(sock_, kOpTouchDown) &&
               write_u32(sock_, static_cast<uint32_t>(x)) &&
               write_u32(sock_, static_cast<uint32_t>(y)) && read_ok_locked();
    }

    bool touch_move(int contact, int x, int y, int pressure) override
    {
        (void)contact;
        (void)pressure;
        std::lock_guard lock(io_);
        return ensure_connected_locked() && write_u8(sock_, kOpTouchMove) &&
               write_u32(sock_, static_cast<uint32_t>(x)) &&
               write_u32(sock_, static_cast<uint32_t>(y)) && read_ok_locked();
    }

    bool touch_up(int contact) override
    {
        std::lock_guard lock(io_);
        return ensure_connected_locked() && write_u8(sock_, kOpTouchUp) &&
               write_u32(sock_, static_cast<uint32_t>(contact)) && read_ok_locked();
    }

    bool click_key(int key) override { return send_key(kOpClickKey, key); }
    bool key_down(int key) override { return send_key(kOpKeyDown, key); }
    bool key_up(int key) override { return send_key(kOpKeyUp, key); }

    bool input_text(const std::string& text) override { return command_string(kOpInputText, text); }

    bool inactive() override
    {
        std::lock_guard lock(io_);
        return ensure_connected_locked() && write_u8(sock_, kOpInactive) && read_ok_locked();
    }

    void* get_info() const override { return nullptr; }

private:
    bool send_key(uint8_t op, int key)
    {
        std::lock_guard lock(io_);
        return ensure_connected_locked() && write_u8(sock_, op) &&
               write_u32(sock_, static_cast<uint32_t>(key)) && read_ok_locked();
    }

    bool command_string(uint8_t op, const std::string& text)
    {
        std::lock_guard lock(io_);
        return ensure_connected_locked() && write_u8(sock_, op) && write_string(sock_, text) &&
               read_ok_locked();
    }

    bool read_ok_locked()
    {
        bool ok = false;
        return read_bool(sock_, ok) && ok;
    }

    bool ensure_connected_locked()
    {
        if (sock_ != INVALID_SOCKET) {
            return true;
        }
        std::string host, port;
        if (!split_endpoint(endpoint_, host, port)) {
            return false;
        }
        if (!wsa_started_) {
            WSADATA data {};
            if (WSAStartup(MAKEWORD(2, 2), &data) != 0) {
                return false;
            }
            wsa_started_ = true;
        }
        addrinfo hints {};
        hints.ai_family = AF_INET;
        hints.ai_socktype = SOCK_STREAM;
        hints.ai_protocol = IPPROTO_TCP;
        addrinfo* result = nullptr;
        if (getaddrinfo(host.c_str(), port.c_str(), &hints, &result) != 0 || !result) {
            return false;
        }
        SOCKET sock = ::socket(result->ai_family, result->ai_socktype, result->ai_protocol);
        if (sock == INVALID_SOCKET) {
            freeaddrinfo(result);
            return false;
        }
        DWORD timeout = 15000;
        setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, reinterpret_cast<const char*>(&timeout), sizeof(timeout));
        setsockopt(sock, SOL_SOCKET, SO_SNDTIMEO, reinterpret_cast<const char*>(&timeout), sizeof(timeout));
        const int rc = ::connect(sock, result->ai_addr, static_cast<int>(result->ai_addrlen));
        freeaddrinfo(result);
        if (rc == SOCKET_ERROR) {
            closesocket(sock);
            return false;
        }
        sock_ = sock;
        if (!write_u8(sock_, kOpConnect)) {
            close_socket();
            return false;
        }
        bool ok = false;
        if (!read_bool(sock_, ok) || !ok) {
            close_socket();
            return false;
        }
        return true;
    }

    void close_socket()
    {
        if (sock_ != INVALID_SOCKET) {
            closesocket(sock_);
            sock_ = INVALID_SOCKET;
        }
    }

    std::string endpoint_;
    mutable std::mutex io_;
    SOCKET sock_ = INVALID_SOCKET;
    bool wsa_started_ = false;
};

class DelegateUnit final : public ControlUnitAPI
{
public:
    DelegateUnit(ControlUnitAPI* child, void (*destroy)(void*), void* raw)
        : child_(child), destroy_(destroy), raw_(raw)
    {
    }

    ~DelegateUnit() override
    {
        if (child_ && destroy_) {
            destroy_(raw_);
        }
    }

    bool connect() override { return child_->connect(); }
    bool connected() const override { return child_->connected(); }
    bool request_uuid(std::string& uuid) override { return child_->request_uuid(uuid); }
    uint64_t get_features() const override { return child_->get_features(); }
    bool start_app(const std::string& intent) override { return child_->start_app(intent); }
    bool stop_app(const std::string& intent) override { return child_->stop_app(intent); }
    bool screencap(cv::Mat& image) override { return child_->screencap(image); }
    bool click(int x, int y) override { return child_->click(x, y); }
    bool swipe(int x1, int y1, int x2, int y2, int duration) override
    {
        return child_->swipe(x1, y1, x2, y2, duration);
    }
    bool touch_down(int contact, int x, int y, int pressure) override
    {
        return child_->touch_down(contact, x, y, pressure);
    }
    bool touch_move(int contact, int x, int y, int pressure) override
    {
        return child_->touch_move(contact, x, y, pressure);
    }
    bool touch_up(int contact) override { return child_->touch_up(contact); }
    bool click_key(int key) override { return child_->click_key(key); }
    bool input_text(const std::string& text) override { return child_->input_text(text); }
    bool key_down(int key) override { return child_->key_down(key); }
    bool key_up(int key) override { return child_->key_up(key); }
    bool inactive() override { return child_->inactive(); }
    void* get_info() const override { return child_->get_info(); }

private:
    ControlUnitAPI* child_ = nullptr;
    void (*destroy_)(void*) = nullptr;
    void* raw_ = nullptr;
};

struct OfficialApi
{
    HMODULE module = nullptr;
    void* (*create)(void*, uint64_t, uint64_t, uint64_t) = nullptr;
    void (*destroy)(void*) = nullptr;
};

OfficialApi& official()
{
    static OfficialApi api;
    static std::once_flag once;
    std::call_once(once, [] {
        const std::wstring self = module_path();
        if (self.empty()) {
            return;
        }
        const auto slash = self.find_last_of(L"\\/");
        const std::wstring path =
            (slash == std::wstring::npos ? std::wstring() : self.substr(0, slash + 1)) +
            L"MaaWin32ControlUnit.official.dll";
        api.module = LoadLibraryW(path.c_str());
        if (!api.module) {
            return;
        }
        api.create = reinterpret_cast<decltype(api.create)>(
            GetProcAddress(api.module, "MaaWin32ControlUnitCreate"));
        api.destroy = reinterpret_cast<decltype(api.destroy)>(
            GetProcAddress(api.module, "MaaWin32ControlUnitDestroy"));
    });
    return api;
}

} // namespace

extern "C" __declspec(dllexport) const char* MaaWin32ControlUnitGetVersion()
{
    return "maago-rdp-controlunit/1";
}

extern "C" __declspec(dllexport) void* MaaWin32ControlUnitCreate(
    void* hwnd,
    uint64_t screencap_method,
    uint64_t mouse_method,
    uint64_t keyboard_method)
{
    char endpoint[512] = {};
    const DWORD endpoint_len =
        GetEnvironmentVariableA("MAAGO_RDP_BRIDGE_ADDR", endpoint, static_cast<DWORD>(sizeof(endpoint)));
    if (endpoint_len > 0 && endpoint_len < sizeof(endpoint)) {
        return static_cast<ControlUnitAPI*>(new RdpUnit(endpoint));
    }

    auto& api = official();
    if (!api.create || !api.destroy) {
        return nullptr;
    }
    void* raw = api.create(hwnd, screencap_method, mouse_method, keyboard_method);
    if (!raw) {
        return nullptr;
    }
    return static_cast<ControlUnitAPI*>(
        new DelegateUnit(static_cast<ControlUnitAPI*>(raw), api.destroy, raw));
}

extern "C" __declspec(dllexport) void MaaWin32ControlUnitDestroy(void* handle)
{
    delete static_cast<ControlUnitAPI*>(handle);
}
