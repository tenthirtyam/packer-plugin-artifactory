d-i auto-install/enable boolean true

d-i debian-installer/language string ${vm_guest_os_language}
d-i debian-installer/country string ${vm_guest_os_country}
d-i debian-installer/locale string ${vm_guest_os_locale}

d-i console-setup/ask_detect boolean false
d-i debconf/frontend select noninteractive

d-i keyboard-configuration/xkb-keymap select ${vm_guest_os_keyboard}
d-i keymap select ${vm_guest_os_keyboard}

choose-mirror-bin mirror/http/proxy string
d-i apt-setup/use_mirror boolean true
d-i base-installer/kernel/override-image string linux-server

d-i clock-setup/utc boolean true
d-i clock-setup/utc-auto boolean true
d-i time/zone string ${vm_guest_os_timezone}

d-i finish-install/reboot_in_progress note

d-i grub-installer/only_debian boolean true
d-i grub-installer/with_other_os boolean true
d-i grub-installer/bootdev string /dev/sda

d-i mirror/country string manual
d-i mirror/http/directory string /debian/
d-i mirror/http/hostname string ${debian_mirror_hostname}
d-i mirror/http/proxy string

d-i partman-efi/non_efi_system boolean true
d-i partman-auto-lvm/guided_size string max
d-i partman-auto/choose_recipe select atomic
d-i partman-auto/method string lvm
d-i partman-lvm/confirm boolean true
d-i partman-lvm/confirm_nooverwrite boolean true
d-i partman-lvm/device_remove_lvm boolean true
d-i partman/choose_partition select finish
d-i partman/confirm boolean true
d-i partman/confirm_nooverwrite boolean true
d-i partman/confirm_write_new_label boolean true

d-i passwd/root-login boolean false
d-i passwd/user-fullname string ${build_username}
d-i passwd/username string ${build_username}
d-i passwd/user-uid string 1000
d-i passwd/user-password password ${build_password}
d-i passwd/user-password-again password ${build_password}
d-i user-setup/allow-password-weak boolean true
d-i user-setup/encrypt-home boolean false

tasksel tasksel/first multiselect standard, ssh-server
d-i pkgsel/include string openssh-server sudo open-vm-tools curl wget
d-i pkgsel/install-language-support boolean false
apt-cdrom-setup apt-setup/cdrom/set-first boolean false
apt-mirror-setup apt-setup/use_mirror boolean true
d-i pkgsel/update-policy select none
d-i pkgsel/upgrade select full-upgrade
popularity-contest popularity-contest/participate boolean false

d-i preseed/late_command string \
  echo "${build_username} ALL=(ALL:ALL) NOPASSWD:ALL" > /target/etc/sudoers.d/${build_username} && \
  chmod 0440 /target/etc/sudoers.d/${build_username}; \
  sed -i '/^deb cdrom:/s/^/#/' /target/etc/apt/sources.list
