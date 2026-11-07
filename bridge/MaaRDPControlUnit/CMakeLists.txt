cmake_minimum_required(VERSION 3.20)
project(MaaRDPControlUnitBridge LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_MSVC_RUNTIME_LIBRARY "MultiThreadedDLL")

# The MaaCore runtime ships opencv_world4_maa.dll. Its import library is
# usually named opencv_world4.lib and comes from the same MaaDeps tree.
set(OPENCV_ROOT "D:/self/maa/MaaDeps/vcpkg/installed/maa-x64-windows" CACHE PATH "MaaDeps OpenCV prefix")
set(MAA_CORE_DIR "D:/self/maa/MaaAssistantArknights/src/MaaCore" CACHE PATH "MaaCore source directory")

add_library(MaaRDPControlUnit SHARED
    MaaRDPControlUnit.cpp
)

target_include_directories(MaaRDPControlUnit PRIVATE
    "${OPENCV_ROOT}/include/opencv4"
)

target_link_libraries(MaaRDPControlUnit PRIVATE
    "${OPENCV_ROOT}/lib/opencv_world4.lib"
    ws2_32
)

# MaaCore's Win32Controller hardcodes the file name and the
# MaaWin32ControlUnit* exports, so this RDP implementation still has to ship
# under the legacy artifact name. The source/project name is the honest one.
set_target_properties(MaaRDPControlUnit PROPERTIES
    OUTPUT_NAME "MaaWin32ControlUnit"
    PREFIX ""
)

add_executable(MaaRDPControlUnitTest test_abi.cpp)
target_include_directories(MaaRDPControlUnitTest PRIVATE
    "${OPENCV_ROOT}/include/opencv4"
    "${MAA_CORE_DIR}"
    "${MAA_CORE_DIR}/../MaaUtils/include"
)
target_link_libraries(MaaRDPControlUnitTest PRIVATE
    "${OPENCV_ROOT}/lib/opencv_world4.lib"
    ws2_32
)
