#!/usr/bin/env bash
set -e

function get_arch() {
    a=$(uname -m)
    case ${a} in
    "x86_64" | "amd64")
        echo "amd64"
        ;;
    "i386" | "i486" | "i586")
        echo "386"
        ;;
    "aarch64" | "arm64")
        echo "arm64"
        ;;
    "armv6l" | "armv7l")
        echo "arm"
        ;;
    "s390x")
        echo "s390x"
        ;;
    "riscv64")
        echo "riscv64"
        ;;
    *)
        echo ${NIL}
        ;;
    esac
}

function get_os() {
    echo $(uname -s | awk '{print tolower($0)}')
}

function package() {
    printf "============Pakcage for %s============\n" $2
    local release=${1}
    local osarch=(${2//_/ })
    local os=${osarch[0]}
    local arch=${osarch[1]}
    local repo=${3}
    local binary="g"
    if [ "${os}" == "windows" ]; then
        binary="g.exe"
    fi

    printf "[1/2] Cross compile@%s_%s\n" ${os} ${arch}
    GOOS=${os} GOARCH=${arch} go build -ldflags "-X github.com/voidint/g/build.UpdateRepo=${repo}" -o "${binary}" .

    printf "[2/2] Package\n"
    if [ ${os} == "windows" ]; then
        zip g${release}.${os}-${arch}.zip ./g.exe
        shasum -a 256 g${release}.${os}-${arch}.zip >>sha256sum.txt
    else
        tar -czv -f g${release}.${os}-${arch}.tar.gz ./g
        shasum -a 256 g${release}.${os}-${arch}.tar.gz >>sha256sum.txt
    fi
}

main() {
    local repo="${G_UPDATE_REPO:-voidint/g}"
    if [[ ! ${repo} =~ ^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9_.-]+$ ]] || [[ ${repo#*/} == "." || ${repo#*/} == ".." ]]; then
        echo "Invalid G_UPDATE_REPO: expected owner/repo" >&2
        return 1
    fi

    export CGO_ENABLED="0"
    export GO111MODULE="on"
    export GOPROXY="https://goproxy.cn,direct"

    local release="1.8.1"

    for item in "darwin_amd64" "darwin_arm64" "linux_386" "linux_amd64" "linux_arm" "linux_arm64" "linux_s390x" "linux_riscv64" "windows_386" "windows_amd64" "windows_arm" "windows_arm64" "freebsd_386" "freebsd_amd64" "freebsd_arm" "freebsd_arm64" "freebsd_riscv64"; do
        package "${release}" "${item}" "${repo}"
    done

    go clean
}

main
