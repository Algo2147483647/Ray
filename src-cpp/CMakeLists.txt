cmake_minimum_required(VERSION 3.15)
project(MyProject)

# 设置C++标准
set(CMAKE_CXX_STANDARD 17)

# 排除CMake生成的文件
file(GLOB_RECURSE SOURCES 
    "*.cpp"
    "*.h"
    "*.hpp"
)

# 过滤掉CMake的编译器检测文件
list(FILTER SOURCES EXCLUDE REGEX ".*CMakeCXXCompilerId\\.cpp$")
list(FILTER SOURCES EXCLUDE REGEX ".*CMakeCCompilerId\\.c$")

add_executable(${PROJECT_NAME} ${SOURCES})

target_include_directories(${PROJECT_NAME} PRIVATE 
    ${CMAKE_CURRENT_SOURCE_DIR}
)