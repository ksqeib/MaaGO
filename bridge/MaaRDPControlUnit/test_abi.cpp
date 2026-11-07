// ABI/protocol smoke test for the RDP control unit. It includes MaaCore's real
// MaaFwControlUnitInterface.h so a vtable mismatch is caught before MaaCore
// ever loads the DLL.

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>

#include <opencv2/core/mat.hpp>

#include "Controller/MaaFwControlUnitInterface.h"

#include <atomic>
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <string>
#include <thread>

namespace
{
constexpr uint8_t kOpConnect = 1;
constexpr uint8_t kOpScreencap = 5;
constexpr uint8_t kOpClick = 6;
constexpr uint8_t kOpKeyDown = 11;
constexpr uint8_t kOpKeyUp = 12;
constexpr uint8_t kOpInputText = 14;
constexpr uint8_t kOpInactive = 15;

bool recv_all(SOCKET s, void* data, size_t len)
{
    auto* p = static_cast<char*>(data);
    while (len > 0) {
        const int n = ::recv(s, p, static_cast<int>(len), 0);
        if (n <= 0) {
            return false;
        }
        p += n;
        len -= static_cast<size_t>(n);
    }
    return true;
}

bool send_all(SOCKET s, const void* data, size_t len)
{
    const auto* p = static_cast<const char*>(data);
    while (len > 0) {
        const int n = ::send(s, p, static_cast<int>(len), 0);
        if (n <= 0) {
            return false;
        }
        p += n;
        len -= static_cast<size_t>(n);
    }
    return true;
}

bool recv_u32(SOCKET s, uint32_t& value)
{
    if (!recv_all(s, &value, sizeof(value))) {
        return false;
    }
    value = ntohl(value);
    return true;
}

bool send_u32(SOCKET s, uint32_t value)
{
    value = htonl(value);
    return send_all(s, &value, sizeof(value));
}

bool send_bool(SOCKET s, bool value)
{
    uint8_t raw = value ? 1 : 0;
    return send_all(s, &raw, sizeof(raw));
}

std::string recv_string(SOCKET s)
{
    uint32_t len = 0;
    if (!recv_u32(s, len) || len > 1024) {
        return {};
    }
    std::string text(len, '\0');
    if (len > 0 && !recv_all(s, text.data(), len)) {
        return {};
    }
    return text;
}
} // namespace

int main(int argc, char** argv)
{
    const char* dll_path = argc > 1 ? argv[1] : "MaaWin32ControlUnit.dll";
    const bool delegate_only = argc > 2 && std::string(argv[2]) == "delegate";

    HMODULE dll = LoadLibraryW(std::wstring(dll_path, dll_path + std::strlen(dll_path)).c_str());
    if (!dll) {
        std::fprintf(stderr, "LoadLibrary failed: %lu\n", GetLastError());
        return 1;
    }
    using CreateFn = void* (*)(void*, uint64_t, uint64_t, uint64_t);
    using DestroyFn = void (*)(void*);
    auto create = reinterpret_cast<CreateFn>(GetProcAddress(dll, "MaaWin32ControlUnitCreate"));
    auto destroy = reinterpret_cast<DestroyFn>(GetProcAddress(dll, "MaaWin32ControlUnitDestroy"));
    if (!create || !destroy) {
        std::fprintf(stderr, "exports missing\n");
        return 1;
    }

    if (delegate_only) {
        // No bridge env: the wrapper must hand the call to the original DLL.
        SetEnvironmentVariableA("MAAGO_RDP_BRIDGE_ADDR", nullptr);
        HWND window = CreateWindowExW(
            0,
            L"STATIC",
            L"MaaGO control-unit delegate test",
            WS_OVERLAPPEDWINDOW,
            0,
            0,
            640,
            360,
            nullptr,
            nullptr,
            GetModuleHandleW(nullptr),
            nullptr);
        auto* unit = create(window, 16, 4, 2);
        const bool ok = unit != nullptr;
        if (unit) {
            destroy(unit);
        }
        if (window) {
            DestroyWindow(window);
        }
        FreeLibrary(dll);
        if (!ok) {
            std::fprintf(stderr, "delegate create returned null\n");
            return 1;
        }
        std::printf("delegate test passed\n");
        return 0;
    }

    WSADATA wsa {};
    if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) {
        std::fprintf(stderr, "WSAStartup failed\n");
        return 1;
    }

    SOCKET listener = ::socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
    if (listener == INVALID_SOCKET) {
        std::fprintf(stderr, "socket failed\n");
        return 1;
    }
    sockaddr_in addr {};
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    addr.sin_port = 0;
    if (::bind(listener, reinterpret_cast<sockaddr*>(&addr), sizeof(addr)) == SOCKET_ERROR ||
        ::listen(listener, 1) == SOCKET_ERROR) {
        std::fprintf(stderr, "bind/listen failed\n");
        return 1;
    }
    int addr_len = sizeof(addr);
    ::getsockname(listener, reinterpret_cast<sockaddr*>(&addr), &addr_len);

    std::atomic<bool> saw_click { false };
    std::atomic<bool> saw_key { false };
    std::atomic<bool> saw_text { false };
    std::thread server([&] {
        SOCKET client = ::accept(listener, nullptr, nullptr);
        if (client == INVALID_SOCKET) {
            return;
        }
        for (;;) {
            uint8_t op = 0;
            if (!recv_all(client, &op, 1)) {
                break;
            }
            if (op == kOpConnect) {
                send_bool(client, true);
            }
            else if (op == kOpScreencap) {
                constexpr uint32_t w = 4, h = 4;
                send_bool(client, true);
                send_u32(client, w);
                send_u32(client, h);
                send_u32(client, w * h * 4);
                std::vector<uint8_t> pixels(w * h * 4, 0x7F);
                send_all(client, pixels.data(), pixels.size());
            }
            else if (op == kOpClick) {
                uint32_t x = 0, y = 0;
                recv_u32(client, x);
                recv_u32(client, y);
                saw_click.store(x == 12 && y == 34);
                send_bool(client, true);
            }
            else if (op == kOpKeyDown || op == kOpKeyUp) {
                uint32_t key = 0;
                recv_u32(client, key);
                saw_key.store(key == 0x1B);
                send_bool(client, true);
            }
            else if (op == kOpInputText) {
                auto text = recv_string(client);
                saw_text.store(text == "abc");
                send_bool(client, true);
            }
            else if (op == kOpInactive) {
                send_bool(client, true);
            }
            else {
                break;
            }
        }
        ::closesocket(client);
        ::closesocket(listener);
    });

    char endpoint[64] = {};
    std::snprintf(endpoint, sizeof(endpoint), "127.0.0.1:%u", ntohs(addr.sin_port));
    SetEnvironmentVariableA("MAAGO_RDP_BRIDGE_ADDR", endpoint);

    auto* unit = static_cast<asst::MaaFwControlUnitAPI*>(create(nullptr, 16, 4, 2));
    if (!unit) {
        std::fprintf(stderr, "create returned null\n");
        return 1;
    }

    bool ok = unit->connect();
    cv::Mat image;
    ok = ok && unit->screencap(image);
    ok = ok && image.cols == 4 && image.rows == 4 && image.type() == CV_8UC4;
    ok = ok && unit->click(12, 34);
    ok = ok && unit->key_down(0x1B);
    ok = ok && unit->key_up(0x1B);
    ok = ok && unit->input_text("abc");
    ok = ok && unit->inactive();
    destroy(unit);

    server.join();
    FreeLibrary(dll);

    if (!ok || !saw_click.load() || !saw_key.load() || !saw_text.load()) {
        std::fprintf(stderr, "ABI/protocol test failed: ok=%d click=%d key=%d text=%d\n",
                     ok, saw_click.load(), saw_key.load(), saw_text.load());
        return 1;
    }
    std::printf("ABI/protocol test passed\n");
    return 0;
}
