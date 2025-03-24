Name:        k2http
Version:     %{__version}
Release:     %{__release}%{?dist}
License:     GNU AGPLv3
URL:         https://github.com/redBorder/k2http
Source0:     %{name}-%{version}.tar.gz

BuildRequires: rsync gcc git
BuildRequires: librd-devel librdkafka-devel
Requires:      librd0 librdkafka

Summary:     RPM used to install k2http in redBorder NG

%define debug_package %{nil}

%description
k2http is a high-performance HTTP service designed for use in the redBorder NG environment.
It provides robust features and is built using the Go programming language.

%prep
%setup -qn %{name}-%{version}

GO_VERSION=$(grep -E '^go [0-9]+\.[0-9]+' go.mod | awk '{print $2}')

if [ -z "$GO_VERSION" ]; then
    echo "Failed to detect Go version from go.mod"
    exit 1
fi

GO_TAR="go${GO_VERSION}.linux-amd64.tar.gz"
GO_DIR="go${GO_VERSION}"

if [ ! -d "$GO_DIR" ]; then
    echo "Downloading Go $GO_VERSION..."
    curl -LO "https://go.dev/dl/$GO_TAR" || { echo "Failed to download Go"; exit 1; }
    tar -xzf "$GO_TAR" || { echo "Failed to extract Go"; exit 1; }
fi

# Set GOROOT to the downloaded Go directory (tarball extracts to "go")
export GOROOT=$(pwd)/go
export PATH=$GOROOT/bin:$PATH

go version
rm -rf go/test
go mod tidy

%build
export GOPATH=${PWD}/gopath
export PATH=$(pwd)/go/bin:${GOPATH}:$PATH

# Verify that the Go version is at least 1.23.0.
go_version=$(go version | awk '{print $3}' | sed 's/go//')
major=$(echo $go_version | cut -d. -f1)
minor=$(echo $go_version | cut -d. -f2)
if [ "$major" -lt 1 ] || { [ "$major" -eq 1 ] && [ "$minor" -lt 23 ]; }; then
    echo "Error: Go 1.23.0 or higher is required. Found go version $go_version."
    exit 1
fi

mkdir -p $GOPATH/src/github.com/redBorder/k2http
rsync -az --exclude=packaging/ --exclude=resources/ --exclude=gopath/ . $GOPATH/src/github.com/redBorder/k2http
cd $GOPATH/src/github.com/redBorder/k2http
make

%install
export GOPATH=${PWD}/gopath
export PATH=$(pwd)/go/bin:${GOPATH}:$PATH
export PKG_CONFIG_PATH=/usr/lib64/pkgconfig
cd $GOPATH/src/github.com/redBorder/k2http
mkdir -p %{buildroot}/usr/bin
prefix=%{buildroot}/usr PKG_CONFIG_PATH=/usr/lib/pkgconfig/ make install
mkdir -p %{buildroot}/usr/share/k2http
mkdir -p %{buildroot}/etc/k2http
install -D -m 644 k2http.service %{buildroot}/usr/lib/systemd/system/k2http.service

%clean
rm -rf %{buildroot}

%pre
getent group k2http >/dev/null || groupadd -r k2http
getent passwd k2http >/dev/null || \
    useradd -r -g k2http -d / -s /sbin/nologin \
    -c "User of k2http service" k2http

%post -p /sbin/ldconfig
%postun -p /sbin/ldconfig

%files
%defattr(755,root,root)
/usr/bin/k2http
/usr/lib/systemd/system/k2http.service

%changelog
* Tue Apr 02 2024 David Vanhoucke <dvanhoucke@redborder.com> - 1.1.7
- Updated dependencies, removed building librdkafka from git, and adjusted config file permissions.
* Tue Feb 08 2022 Vicente Mesa <vimesa@redborder.com> - 1.0.0
- Initial spec version.

