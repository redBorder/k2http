Name: k2http
Version: %{__version}
Release: %{__release}%{?dist}

License: AGPL 3.0
URL: https://github.com/redBorder/k2http
Source0: %{name}-%{version}.tar.gz

BuildRequires: go = 1.6.3
BuildRequires: glide rsync gcc git
BuildRequires:	rsync mlocate

Summary: rpm used to install k2http in a redborder ng
Group:   Development/Libraries/Go

%description
%{summary}

%prep
%setup -qn %{name}-%{version}

%build

export GOPATH=${PWD}/gopath
export PATH=${GOPATH}:${PATH}

mkdir -p $GOPATH/src/github.com/redBorder/k2http
rsync -az --exclude=packaging/ --exclude=resources/ --exclude=gopath/ ./ $GOPATH/src/github.com/redBorder/k2http
cd $GOPATH/src/github.com/redBorder/k2http
ls -lsa
make

%install
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/etc/k2http

export PARENT_BUILD=${PWD}
export GOPATH=${PWD}/gopath
export PATH=${GOPATH}:${PATH}
pushd $GOPATH/src/github.com/redBorder/k2http
echo "Making make install.." 
prefix=%{buildroot}/usr make install
popd
cp -f resources/files/config.yml.default %{buildroot}/etc/k2http/
install -D -m 0644 resources/systemd/k2http.service %{buildroot}/usr/lib/systemd/system/k2http.service


%clean
rm -rf %{buildroot}

%pre
getent group k2http >/dev/null || groupadd -r k2http
getent passwd k2http >/dev/null || \
    useradd -r -g k2http -d / -s /sbin/nologin \
    -c "User of k2http service" k2http
exit 0

%post
[ ! -f /etc/sysconfig/k2http ] && cp /etc/sysconfig/k2http.default /etc/sysconfig/k2http
systemctl daemon-reload
/usr/lib/redborder/bin/rb_rubywrapper.sh -c
mkdir -p /var/log/k2http
/sbin/ldconfig

%postun -p /sbin/ldconfig

%files
%defattr(0755,root,root)
/usr/bin/rb_register
%defattr(644,root,root)
/usr/lib/systemd/system/k2http.service
/etc/sysconfig/k2http.default
/etc/chef
%defattr(755,root,root)
/etc/k2http/config.yml.default
%doc

%changelog
* Mon Mar 28 2022 Miguel Negron <manegron@redborder.com> - 0.0.1
- First spec version
