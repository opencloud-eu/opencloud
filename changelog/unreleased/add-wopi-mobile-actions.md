Enhancement: Support mobileView and mobileEdit WOPI actions

The collaboration service now parses the mobileView and mobileEdit actions from
the WOPI discovery and serves them to mobile browsers, so mobile users get the
mobile optimized editor instead of the desktop one. If the document server does
not announce a mobile action, the previous desktop URL is used as before. This
only affects OnlyOffice compatible document servers, Collabora adapts its UI on
its own. The behaviour can be turned off with COLLABORATION_WOPI_ENABLE_MOBILE.

https://github.com/opencloud-eu/opencloud/issues/3167
https://github.com/opencloud-eu/opencloud/pull/3292
