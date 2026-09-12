include $(TOPDIR)/rules.mk

PKG_NAME:=sdm-core
PKG_VERSION:=4.1.0
PKG_RELEASE:=1

PKG_SOURCE:=$(PKG_NAME)-$(PKG_VERSION).tar.gz
PKG_SOURCE_URL:=https://codeload.github.com/HZ-PRE/kuailei-core/tar.gz/v$(PKG_VERSION)?
PKG_HASH:=8a877268eaea3dc52f300ddfd2f702ed1c4e37f75ddea5fcecc27cfefc849f85

PKG_LICENSE:=https://github.com/sdm/sdm-app/edit/main/LICENSE.md
PKG_LICENSE_FILES:=LICENSE
PKG_MAINTAINER:=Sdm <wrt@sdm.com>

PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_PARALLEL:=1
PKG_BUILD_FLAGS:=no-mips16

GO_PKG:=github.com/HZ-PRE/kuailei-core
GO_PKG_BUILD_PKG:=$(GO_PKG)/cli

GO_PKG_LDFLAGS_X:=$(GO_PKG)/constant.Version=$(PKG_VERSION)

include $(INCLUDE_DIR)/package.mk
include ../../lang/golang/golang-package.mk

define Package/sdm-core
  TITLE:=Sdm universal proxy platform
  SECTION:=net
  CATEGORY:=Network
  URL:=https://sdm.com/
  DEPENDS:=$(GO_ARCH_DEPENDS) +ca-bundle +kmod-inet-diag +kmod-tun
  USERID:=sdm=15566:sdm=15566
endef

define Package/sdm-core/description
  Sdm is a universal proxy platform which supports hysteria, SOCKS, Shadowsocks,
  ShadowTLS, Tor, trojan, VLess, VMess, WireGuard and so on.
endef

GO_PKG_TAGS:=with_gvisor,with_quic,with_wireguard,with_ech,with_utls,with_clash_api,with_grpc,with_low_memory,with_conntrack

define Package/sdm-core/conffiles
/etc/config/sdm
/etc/sdm/
endef

define Package/sdm-core/install

	$(INSTALL_DIR) $(1)/usr/bin/
	$(INSTALL_BIN) $(GO_PKG_BUILD_BIN_DIR)/cli $(1)/usr/bin/sdm-cli

	$(INSTALL_DIR) $(1)/etc/sdm
	$(INSTALL_DATA) ./files/sdm.json $(1)/etc/sdm/appconfig.json

	$(INSTALL_DIR) $(1)/etc/config/
	$(INSTALL_CONF) ./files/sdm.conf $(1)/etc/config/sdm

	$(INSTALL_DIR) $(1)/etc/init.d/
	$(INSTALL_BIN) ./files/sdm.init $(1)/etc/init.d/sdm

	$(CP) ./files/webui $(1)/usr/bin/
endef

$(eval $(call BuildPackage,sdm-core))
