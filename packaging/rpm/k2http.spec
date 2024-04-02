Name:    k2http
Version: %{__version}
Release: %{__release}%{?dist}

License: GNU AGPLv3
URL: https://github.com/redBorder/k2http
Source0: %{name}-%{version}.tar.gz

BuildRequires: go rsync gcc git
BuildRequires: librd-devel librdkafka-devel
Requires: librd0 librdkafka 

Summary: rpm used to install k2http in redborder ng
Group: Development/Libraries/Go

%description
%{summary}

%define debug_package %{nil}

%prep
%setup -qn %{name}-%{version}

%build
export GOPATH=${PWD}/gopath
export PATH=${GOPATH}:${PATH}

mkdir -p $GOPATH/src/github.com/redBorder/k2http
rsync -az --exclude=packaging/ --exclude=resources/ --exclude=gopath/ ./ $GOPATH/src/github.com/redBorder/k2http
cd $GOPATH/src/github.com/redBorder/k2http
make

%install
export PARENT_BUILD=${PWD}
export GOPATH=${PWD}/gopath
export PATH=${GOPATH}:${PATH}
export PKG_CONFIG_PATH=/usr/lib64/pkgconfig
cd $GOPATH/src/github.com/redBorder/k2http
mkdir -p %{buildroot}/usr/bin
prefix=%{buildroot}/usr PKG_CONFIG_PATH=/usr/lib/pkgconfig/ make install
mkdir -p %{buildroot}/usr/share/k2http
mkdir -p %{buildroot}/etc/k2http
install -D -m 644 k2http.service %{buildroot}/usr/lib/systemd/system/k2http.service
#install -D -m 644 packaging/rpm/config.yml %{buildroot}/usr/share/k2http

%clean
rm -rf %{buildroot}

%pre
getent group k2http >/dev/null || groupadd -r k2http
getent passwd k2http >/dev/null || \
    useradd -r -g k2http -d / -s /sbin/nologin \
    -c "User of k2http service" k2http
exit 0

%post -p /sbin/ldconfig
%postun -p /sbin/ldconfig

%files
%defattr(755,root,root)
/usr/bin/k2http
#%defattr(644,root,root)
#/usr/share/k2http/config.yml
/usr/lib/systemd/system/k2http.service

%changelog
* Tue Apr 02 2024 David Vanhoucke <dvanhoucke@redborder.com> - 1.1.6
- Dependencies changed, erased the building of librdkafka from git repo and no longer giving permissions to config.yml
* Tue Feb 08 2022 Vicente Mesa <vimesa@redborder.com> - 1.0.0
- first spec version
