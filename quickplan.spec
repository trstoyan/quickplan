Name:           quickplan
Version:        0.1.0
Release:        1%{?dist}
Summary:        Fast CLI task manager with project support
License:        MIT
URL:            https://github.com/trstoyan/quickplan
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.21

%description
QuickPlan is a terminal-based task manager that lets you organize
tasks into named projects with vim-inspired selection menus.

%prep
%setup -q

%build
export CGO_ENABLED=0
export GOOS=linux
go build -ldflags "-X main.version=%{version}" -o %{name}

%install
mkdir -p %{buildroot}%{_bindir}
install -m 755 %{name} %{buildroot}%{_bindir}/%{name}

%files
%{_bindir}/%{name}

%changelog
* Mon Jan 01 2024 Stoyan TR <stoyantr@icloud.com> - 0.1.0-1
- Initial release
