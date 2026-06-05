package jmap

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/google/go-cmp/cmp"
	dockernetwork "github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"

	"github.com/gorilla/websocket"
	"github.com/tidwall/pretty"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/brianvoe/gofakeit/v7"
	pw "github.com/sethvargo/go-password/password"

	"github.com/opencloud-eu/opencloud/pkg/jscontact"
	oclog "github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

const (
	EnableTypes = false

	// Wireshark = "/usr/bin/wireshark"
	Wireshark = ""
)

type User struct {
	name        string
	description string
	alias       string
	password    string
	email       string
}

func userpassword() string {
	password, err := pw.Generate(10+rand.Intn(28), 2, 0, false, true)
	if err != nil {
		panic(err)
	}
	return password
}

func mkuser(name string, description string, alias string) User {
	parts := strings.Split(alias, "@")
	return User{name: name, description: description, alias: alias, email: name + "@" + parts[1], password: userpassword()}
}

var (
	users = [...]User{
		mkuser("cdrummer", "Camina Drummer", "camina.drummer@opa.org"),
		mkuser("aburton", "Amos Burton", "amos.burton@earth.gov"),
		mkuser("jholden", "James Holden", "james.holden@earth.gov"),
		mkuser("adawes", "Anderson Dawes", "anderson.dawes@opa.org"),
		mkuser("nnagata", "Naomi Nagata", "naomi.nagata@opa.org"),
		mkuser("kashford", "Klaes Ashford", "klaes.ashford@opa.org"),
		mkuser("fjohnson", "Fred Johnson", "fred.johnson@opa.org"),
		mkuser("cavasarala", "Chrisjen Avasarala}", "chrissy@earth.gov"),
		mkuser("bdraper", "Roberta Draper", "bobby@mars.mil"),
	}
)

const (
	stalwartImage      = "ghcr.io/stalwartlabs/stalwart:v0.16.7-alpine"
	httpPort           = "8080"
	imapsPort          = "993"
	jsonConfigTemplate = `{"@type":"RocksDb","path":"/var/lib/stalwart/","blobSize":16834,"bufferSize":134217728,"poolWorkers":null}`
	cliDockerfile      = `
FROM alpine:latest
ARG VERSION="1.0.8"
ARG ARCH="x86_64"
RUN apk add --no-cache curl tar xz && curl --proto '=https' --tlsv1.2 -LsSf https://github.com/stalwartlabs/cli/releases/download/v${VERSION}/stalwart-cli-${ARCH}-unknown-linux-musl.tar.xz | tar xJf - --strip-components=1 -C /usr/local/bin/ && rm /usr/local/bin/*.md
CMD ["/usr/local/bin/stalwart-cli"]
`
)

var (
	dumpTemplate = []string{
		`{"@type":"destroy","object":"Tracer"}`,
		`{"@type":"destroy","object":"NetworkListener"}`,
		`{"@type":"destroy","object":"DnsServer"}`,
		`{"@type":"destroy","object":"Certificate"}`,
		`{"@type":"destroy","object":"Tenant"}`,
		`{"@type":"destroy","object":"Role"}`,
		`{"@type":"destroy","object":"AcmeProvider"}`,
		`{"@type":"destroy","object":"Directory"}`,
		`{"@type":"destroy","object":"Domain"}`,
		`{"@type":"destroy","object":"Account"}`,
		`{"@type":"destroy","object":"DkimSignature"}`,
		`{"@type":"create","object":"Tracer","value":{"tracer-iugg9zgwaaaa":{"@type":"Stdout","eventsPolicy":"exclude","enable":true,"multiline":false,"level":"trace","lossy":false,"buffered":true,"ansi":false,"events":{"store.data-write":true,"eval.result":true,"store.cache-hit":true,"store.cache-miss":true,"store.cache-stale":true,"store.cache-update":true,"store.data-iterate":true,"spam.rules-updated":true,"resource.download-external":true,"task-manager.task-acquired":true,"task-manager.scheduler-started":true,"task-manager.manager-started":true}}}}`,
		`{"@type":"create","object":"NetworkListener","value":{"networklistener-iughfwjyahqb":{"socketNoDelay":true,"socketReusePort":true,"tlsDisableCipherSuites":{},"socketReuseAddress":true,"name":"http","bind":{"[::]:8080":true},"socketTosV4":null,"socketTtl":null,"maxConnections":8192,"socketBacklog":1024,"socketReceiveBufferSize":null,"overrideProxyTrustedNetworks":{},"socketSendBufferSize":null,"tlsIgnoreClientOrder":true,"tlsImplicit":false,"tlsTimeout":60000,"useTls":true,"tlsDisableProtocols":{},"protocol":"http"},"networklistener-iughfwjyahab":{"socketNoDelay":true,"socketReusePort":true,"tlsDisableCipherSuites":{},"socketReuseAddress":true,"name":"https","bind":{"[::]:443":true},"socketTosV4":null,"socketTtl":null,"maxConnections":8192,"socketBacklog":1024,"socketReceiveBufferSize":null,"overrideProxyTrustedNetworks":{},"socketSendBufferSize":null,"tlsIgnoreClientOrder":true,"tlsImplicit":true,"tlsTimeout":60000,"useTls":true,"tlsDisableProtocols":{},"protocol":"http"},"networklistener-iughfwjyagqb":{"socketNoDelay":true,"socketReusePort":true,"tlsDisableCipherSuites":{},"socketReuseAddress":true,"name":"sieve","bind":{"[::]:4190":true},"socketTosV4":null,"socketTtl":null,"maxConnections":8192,"socketBacklog":1024,"socketReceiveBufferSize":null,"overrideProxyTrustedNetworks":{},"socketSendBufferSize":null,"tlsIgnoreClientOrder":true,"tlsImplicit":false,"tlsTimeout":60000,"useTls":true,"tlsDisableProtocols":{},"protocol":"manageSieve"},"networklistener-iughfwjyagab":{"socketNoDelay":true,"socketReusePort":true,"tlsDisableCipherSuites":{},"socketReuseAddress":true,"name":"pop3s","bind":{"[::]:995":true},"socketTosV4":null,"socketTtl":null,"maxConnections":8192,"socketBacklog":1024,"socketReceiveBufferSize":null,"overrideProxyTrustedNetworks":{},"socketSendBufferSize":null,"tlsIgnoreClientOrder":true,"tlsImplicit":true,"tlsTimeout":60000,"useTls":true,"tlsDisableProtocols":{},"protocol":"pop3"},"networklistener-iughfwjyafqb":{"socketNoDelay":true,"socketReusePort":true,"tlsDisableCipherSuites":{},"socketReuseAddress":true,"name":"imaps","bind":{"[::]:993":true},"socketTosV4":null,"socketTtl":null,"maxConnections":8192,"socketBacklog":1024,"socketReceiveBufferSize":null,"overrideProxyTrustedNetworks":{},"socketSendBufferSize":null,"tlsIgnoreClientOrder":true,"tlsImplicit":true,"tlsTimeout":60000,"useTls":true,"tlsDisableProtocols":{},"protocol":"imap"},"networklistener-iughfwjwafab":{"socketNoDelay":true,"socketReusePort":true,"tlsDisableCipherSuites":{},"socketReuseAddress":true,"name":"submissions","bind":{"[::]:465":true},"socketTosV4":null,"socketTtl":null,"maxConnections":8192,"socketBacklog":1024,"socketReceiveBufferSize":null,"overrideProxyTrustedNetworks":{},"socketSendBufferSize":null,"tlsIgnoreClientOrder":true,"tlsImplicit":true,"tlsTimeout":60000,"useTls":true,"tlsDisableProtocols":{},"protocol":"smtp"},"networklistener-iughfwjwaeqb":{"socketNoDelay":true,"socketReusePort":true,"tlsDisableCipherSuites":{},"socketReuseAddress":true,"name":"smtp","bind":{"[::]:25":true},"socketTosV4":null,"socketTtl":null,"maxConnections":8192,"socketBacklog":1024,"socketReceiveBufferSize":null,"overrideProxyTrustedNetworks":{},"socketSendBufferSize":null,"tlsIgnoreClientOrder":true,"tlsImplicit":false,"tlsTimeout":60000,"useTls":true,"tlsDisableProtocols":{},"protocol":"smtp"}}}`,
		`{"@type":"create","object":"Role","value":{"role-e":{"disabledPermissions":{},"memberTenantId":null,"enabledPermissions":{"authenticate":true,"authenticateWithAlias":true,"interactAi":true,"impersonate":true,"unlimitedRequests":true,"unlimitedUploads":true,"fetchAnyBlob":true,"oAuthClientRegistration":true,"oAuthClientOverride":true,"liveTracing":true,"liveMetrics":true,"liveDeliveryTest":true,"sysAccountGet":true,"sysAccountCreate":true,"sysAccountUpdate":true,"sysAccountDestroy":true,"sysAccountQuery":true,"sysAccountPasswordGet":true,"sysAccountPasswordUpdate":true,"sysAccountSettingsGet":true,"sysAccountSettingsUpdate":true,"sysAcmeProviderGet":true,"sysAcmeProviderCreate":true,"sysAcmeProviderUpdate":true,"sysAcmeProviderDestroy":true,"sysAcmeProviderQuery":true,"actionReloadSettings":true,"actionReloadTlsCertificates":true,"actionReloadLookupStores":true,"actionReloadBlockedIps":true,"actionUpdateApps":true,"actionTroubleshootDmarc":true,"actionClassifySpam":true,"actionInvalidateCaches":true,"actionInvalidateNegativeCaches":true,"actionPauseMtaQueue":true,"actionResumeMtaQueue":true,"sysActionGet":true,"sysActionCreate":true,"sysActionUpdate":true,"sysActionDestroy":true,"sysActionQuery":true,"sysAddressBookGet":true,"sysAddressBookUpdate":true,"sysAiModelGet":true,"sysAiModelCreate":true,"sysAiModelUpdate":true,"sysAiModelDestroy":true,"sysAiModelQuery":true,"sysAlertGet":true,"sysAlertCreate":true,"sysAlertUpdate":true,"sysAlertDestroy":true,"sysAlertQuery":true,"sysAllowedIpGet":true,"sysAllowedIpCreate":true,"sysAllowedIpUpdate":true,"sysAllowedIpDestroy":true,"sysAllowedIpQuery":true,"sysApiKeyGet":true,"sysApiKeyCreate":true,"sysApiKeyUpdate":true,"sysApiKeyDestroy":true,"sysApiKeyQuery":true,"sysAppPasswordGet":true,"sysAppPasswordCreate":true,"sysAppPasswordUpdate":true,"sysAppPasswordDestroy":true,"sysAppPasswordQuery":true,"sysApplicationGet":true,"sysApplicationCreate":true,"sysApplicationUpdate":true,"sysApplicationDestroy":true,"sysApplicationQuery":true,"sysArchivedItemGet":true,"sysArchivedItemCreate":true,"sysArchivedItemUpdate":true,"sysArchivedItemDestroy":true,"sysArchivedItemQuery":true,"sysArfExternalReportGet":true,"sysArfExternalReportCreate":true,"sysArfExternalReportUpdate":true,"sysArfExternalReportDestroy":true,"sysArfExternalReportQuery":true,"sysAsnGet":true,"sysAsnUpdate":true,"sysAuthenticationGet":true,"sysAuthenticationUpdate":true,"sysBlobStoreGet":true,"sysBlobStoreUpdate":true,"sysBlockedIpGet":true,"sysBlockedIpCreate":true,"sysBlockedIpUpdate":true,"sysBlockedIpDestroy":true,"sysBlockedIpQuery":true,"sysBootstrapGet":true,"sysBootstrapUpdate":true,"sysCacheGet":true,"sysCacheUpdate":true,"sysCalendarGet":true,"sysCalendarUpdate":true,"sysCalendarAlarmGet":true,"sysCalendarAlarmUpdate":true,"sysCalendarSchedulingGet":true,"sysCalendarSchedulingUpdate":true,"sysCertificateGet":true,"sysCertificateCreate":true,"sysCertificateUpdate":true,"sysCertificateDestroy":true,"sysCertificateQuery":true,"sysClusterNodeGet":true,"sysClusterNodeCreate":true,"sysClusterNodeUpdate":true,"sysClusterNodeDestroy":true,"sysClusterNodeQuery":true,"sysClusterRoleGet":true,"sysClusterRoleCreate":true,"sysClusterRoleUpdate":true,"sysClusterRoleDestroy":true,"sysClusterRoleQuery":true,"sysCoordinatorGet":true,"sysCoordinatorUpdate":true,"sysDataRetentionGet":true,"sysDataRetentionUpdate":true,"sysDataStoreGet":true,"sysDataStoreUpdate":true,"sysDirectoryGet":true,"sysDirectoryCreate":true,"sysDirectoryUpdate":true,"sysDirectoryDestroy":true,"sysDirectoryQuery":true,"sysDkimReportSettingsGet":true,"sysDkimReportSettingsUpdate":true,"sysDkimSignatureGet":true,"sysDkimSignatureCreate":true,"sysDkimSignatureUpdate":true,"sysDkimSignatureDestroy":true,"sysDkimSignatureQuery":true,"sysDmarcExternalReportGet":true,"sysDmarcExternalReportCreate":true,"sysDmarcExternalReportUpdate":true,"sysDmarcExternalReportDestroy":true,"sysDmarcExternalReportQuery":true,"sysDmarcInternalReportGet":true,"sysDmarcInternalReportCreate":true,"sysDmarcInternalReportUpdate":true,"sysDmarcInternalReportDestroy":true,"sysDmarcInternalReportQuery":true,"sysDmarcReportSettingsGet":true,"sysDmarcReportSettingsUpdate":true,"sysDnsResolverGet":true,"sysDnsResolverUpdate":true,"sysDnsServerGet":true,"sysDnsServerCreate":true,"sysDnsServerUpdate":true,"sysDnsServerDestroy":true,"sysDnsServerQuery":true,"sysDomainGet":true,"sysDomainCreate":true,"sysDomainUpdate":true,"sysDomainDestroy":true,"sysDomainQuery":true,"sysDsnReportSettingsGet":true,"sysDsnReportSettingsUpdate":true,"sysEmailGet":true,"sysEmailUpdate":true,"sysEnterpriseGet":true,"sysEnterpriseUpdate":true,"sysEventTracingLevelGet":true,"sysEventTracingLevelCreate":true,"sysEventTracingLevelUpdate":true,"sysEventTracingLevelDestroy":true,"sysEventTracingLevelQuery":true,"sysFileStorageGet":true,"sysFileStorageUpdate":true,"sysHttpGet":true,"sysHttpUpdate":true,"sysHttpFormGet":true,"sysHttpFormUpdate":true,"sysHttpLookupGet":true,"sysHttpLookupCreate":true,"sysHttpLookupUpdate":true,"sysHttpLookupDestroy":true,"sysHttpLookupQuery":true,"sysImapGet":true,"sysImapUpdate":true,"sysInMemoryStoreGet":true,"sysInMemoryStoreUpdate":true,"sysJmapGet":true,"sysJmapUpdate":true,"sysLogGet":true,"sysLogCreate":true,"sysLogUpdate":true,"sysLogDestroy":true,"sysLogQuery":true,"sysMailingListGet":true,"sysMailingListCreate":true,"sysMailingListUpdate":true,"sysMailingListDestroy":true,"sysMailingListQuery":true,"sysMaskedEmailGet":true,"sysMaskedEmailCreate":true,"sysMaskedEmailUpdate":true,"sysMaskedEmailDestroy":true,"sysMaskedEmailQuery":true,"sysMemoryLookupKeyGet":true,"sysMemoryLookupKeyCreate":true,"sysMemoryLookupKeyUpdate":true,"sysMemoryLookupKeyDestroy":true,"sysMemoryLookupKeyQuery":true,"sysMemoryLookupKeyValueGet":true,"sysMemoryLookupKeyValueCreate":true,"sysMemoryLookupKeyValueUpdate":true,"sysMemoryLookupKeyValueDestroy":true,"sysMemoryLookupKeyValueQuery":true,"sysMetricGet":true,"sysMetricCreate":true,"sysMetricUpdate":true,"sysMetricDestroy":true,"sysMetricQuery":true,"sysMetricsGet":true,"sysMetricsUpdate":true,"sysMetricsStoreGet":true,"sysMetricsStoreUpdate":true,"sysMtaConnectionStrategyGet":true,"sysMtaConnectionStrategyCreate":true,"sysMtaConnectionStrategyUpdate":true,"sysMtaConnectionStrategyDestroy":true,"sysMtaConnectionStrategyQuery":true,"sysMtaDeliveryScheduleGet":true,"sysMtaDeliveryScheduleCreate":true,"sysMtaDeliveryScheduleUpdate":true,"sysMtaDeliveryScheduleDestroy":true,"sysMtaDeliveryScheduleQuery":true,"sysMtaExtensionsGet":true,"sysMtaExtensionsUpdate":true,"sysMtaHookGet":true,"sysMtaHookCreate":true,"sysMtaHookUpdate":true,"sysMtaHookDestroy":true,"sysMtaHookQuery":true,"sysMtaInboundSessionGet":true,"sysMtaInboundSessionUpdate":true,"sysMtaInboundThrottleGet":true,"sysMtaInboundThrottleCreate":true,"sysMtaInboundThrottleUpdate":true,"sysMtaInboundThrottleDestroy":true,"sysMtaInboundThrottleQuery":true,"sysMtaMilterGet":true,"sysMtaMilterCreate":true,"sysMtaMilterUpdate":true,"sysMtaMilterDestroy":true,"sysMtaMilterQuery":true,"sysMtaOutboundStrategyGet":true,"sysMtaOutboundStrategyUpdate":true,"sysMtaOutboundThrottleGet":true,"sysMtaOutboundThrottleCreate":true,"sysMtaOutboundThrottleUpdate":true,"sysMtaOutboundThrottleDestroy":true,"sysMtaOutboundThrottleQuery":true,"sysMtaQueueQuotaGet":true,"sysMtaQueueQuotaCreate":true,"sysMtaQueueQuotaUpdate":true,"sysMtaQueueQuotaDestroy":true,"sysMtaQueueQuotaQuery":true,"sysMtaRouteGet":true,"sysMtaRouteCreate":true,"sysMtaRouteUpdate":true,"sysMtaRouteDestroy":true,"sysMtaRouteQuery":true,"sysMtaStageAuthGet":true,"sysMtaStageAuthUpdate":true,"sysMtaStageConnectGet":true,"sysMtaStageConnectUpdate":true,"sysMtaStageDataGet":true,"sysMtaStageDataUpdate":true,"sysMtaStageEhloGet":true,"sysMtaStageEhloUpdate":true,"sysMtaStageMailGet":true,"sysMtaStageMailUpdate":true,"sysMtaStageRcptGet":true,"sysMtaStageRcptUpdate":true,"sysMtaStsGet":true,"sysMtaStsUpdate":true,"sysMtaTlsStrategyGet":true,"sysMtaTlsStrategyCreate":true,"sysMtaTlsStrategyUpdate":true,"sysMtaTlsStrategyDestroy":true,"sysMtaTlsStrategyQuery":true,"sysMtaVirtualQueueGet":true,"sysMtaVirtualQueueCreate":true,"sysMtaVirtualQueueUpdate":true,"sysMtaVirtualQueueDestroy":true,"sysMtaVirtualQueueQuery":true,"sysNetworkListenerGet":true,"sysNetworkListenerCreate":true,"sysNetworkListenerUpdate":true,"sysNetworkListenerDestroy":true,"sysNetworkListenerQuery":true,"sysOAuthClientGet":true,"sysOAuthClientCreate":true,"sysOAuthClientUpdate":true,"sysOAuthClientDestroy":true,"sysOAuthClientQuery":true,"sysOidcProviderGet":true,"sysOidcProviderUpdate":true,"sysPublicKeyGet":true,"sysPublicKeyCreate":true,"sysPublicKeyUpdate":true,"sysPublicKeyDestroy":true,"sysPublicKeyQuery":true,"sysQueuedMessageGet":true,"sysQueuedMessageCreate":true,"sysQueuedMessageUpdate":true,"sysQueuedMessageDestroy":true,"sysQueuedMessageQuery":true,"sysReportSettingsGet":true,"sysReportSettingsUpdate":true,"sysRoleGet":true,"sysRoleCreate":true,"sysRoleUpdate":true,"sysRoleDestroy":true,"sysRoleQuery":true,"sysSearchGet":true,"sysSearchUpdate":true,"sysSearchStoreGet":true,"sysSearchStoreUpdate":true,"sysSecurityGet":true,"sysSecurityUpdate":true,"sysSenderAuthGet":true,"sysSenderAuthUpdate":true,"sysSharingGet":true,"sysSharingUpdate":true,"sysSieveSystemInterpreterGet":true,"sysSieveSystemInterpreterUpdate":true,"sysSieveSystemScriptGet":true,"sysSieveSystemScriptCreate":true,"sysSieveSystemScriptUpdate":true,"sysSieveSystemScriptDestroy":true,"sysSieveSystemScriptQuery":true,"sysSieveUserInterpreterGet":true,"sysSieveUserInterpreterUpdate":true,"sysSieveUserScriptGet":true,"sysSieveUserScriptCreate":true,"sysSieveUserScriptUpdate":true,"sysSieveUserScriptDestroy":true,"sysSieveUserScriptQuery":true,"sysSpamClassifierGet":true,"sysSpamClassifierUpdate":true,"sysSpamDnsblServerGet":true,"sysSpamDnsblServerCreate":true,"sysSpamDnsblServerUpdate":true,"sysSpamDnsblServerDestroy":true,"sysSpamDnsblServerQuery":true,"sysSpamDnsblSettingsGet":true,"sysSpamDnsblSettingsUpdate":true,"sysSpamFileExtensionGet":true,"sysSpamFileExtensionCreate":true,"sysSpamFileExtensionUpdate":true,"sysSpamFileExtensionDestroy":true,"sysSpamFileExtensionQuery":true,"sysSpamLlmGet":true,"sysSpamLlmUpdate":true,"sysSpamPyzorGet":true,"sysSpamPyzorUpdate":true,"sysSpamRuleGet":true,"sysSpamRuleCreate":true,"sysSpamRuleUpdate":true,"sysSpamRuleDestroy":true,"sysSpamRuleQuery":true,"sysSpamSettingsGet":true,"sysSpamSettingsUpdate":true,"sysSpamTagGet":true,"sysSpamTagCreate":true,"sysSpamTagUpdate":true,"sysSpamTagDestroy":true,"sysSpamTagQuery":true,"sysSpamTrainingSampleGet":true,"sysSpamTrainingSampleCreate":true,"sysSpamTrainingSampleUpdate":true,"sysSpamTrainingSampleDestroy":true,"sysSpamTrainingSampleQuery":true,"sysSpfReportSettingsGet":true,"sysSpfReportSettingsUpdate":true,"sysStoreLookupGet":true,"sysStoreLookupCreate":true,"sysStoreLookupUpdate":true,"sysStoreLookupDestroy":true,"sysStoreLookupQuery":true,"sysSystemSettingsGet":true,"sysSystemSettingsUpdate":true,"taskIndexDocument":true,"taskUnindexDocument":true,"taskIndexTrace":true,"taskCalendarAlarmEmail":true,"taskCalendarAlarmNotification":true,"taskCalendarItipMessage":true,"taskMergeThreads":true,"taskDmarcReport":true,"taskTlsReport":true,"taskRestoreArchivedItem":true,"taskDestroyAccount":true,"taskAccountMaintenance":true,"taskTenantMaintenance":true,"taskStoreMaintenance":true,"taskSpamFilterMaintenance":true,"taskAcmeRenewal":true,"taskDkimManagement":true,"taskDnsManagement":true,"sysTaskGet":true,"sysTaskCreate":true,"sysTaskUpdate":true,"sysTaskDestroy":true,"sysTaskQuery":true,"sysTaskManagerGet":true,"sysTaskManagerUpdate":true,"sysTenantGet":true,"sysTenantCreate":true,"sysTenantUpdate":true,"sysTenantDestroy":true,"sysTenantQuery":true,"sysTlsExternalReportGet":true,"sysTlsExternalReportCreate":true,"sysTlsExternalReportUpdate":true,"sysTlsExternalReportDestroy":true,"sysTlsExternalReportQuery":true,"sysTlsInternalReportGet":true,"sysTlsInternalReportCreate":true,"sysTlsInternalReportUpdate":true,"sysTlsInternalReportDestroy":true,"sysTlsInternalReportQuery":true,"sysTlsReportSettingsGet":true,"sysTlsReportSettingsUpdate":true,"sysTraceGet":true,"sysTraceCreate":true,"sysTraceUpdate":true,"sysTraceDestroy":true,"sysTraceQuery":true,"sysTracerGet":true,"sysTracerCreate":true,"sysTracerUpdate":true,"sysTracerDestroy":true,"sysTracerQuery":true,"sysTracingStoreGet":true,"sysTracingStoreUpdate":true,"sysWebDavGet":true,"sysWebDavUpdate":true,"sysWebHookGet":true,"sysWebHookCreate":true,"sysWebHookUpdate":true,"sysWebHookDestroy":true,"sysWebHookQuery":true},"description":"System Administrator","roleIds":{}},"role-d":{"disabledPermissions":{},"memberTenantId":null,"enabledPermissions":{"authenticate":true,"authenticateWithAlias":true,"interactAi":true,"fetchAnyBlob":true,"liveDeliveryTest":true,"sysAccountGet":true,"sysAccountCreate":true,"sysAccountUpdate":true,"sysAccountDestroy":true,"sysAccountQuery":true,"sysAcmeProviderGet":true,"sysAcmeProviderCreate":true,"sysAcmeProviderUpdate":true,"sysAcmeProviderDestroy":true,"sysAcmeProviderQuery":true,"sysDkimSignatureGet":true,"sysDkimSignatureCreate":true,"sysDkimSignatureUpdate":true,"sysDkimSignatureDestroy":true,"sysDkimSignatureQuery":true,"sysDnsServerGet":true,"sysDnsServerCreate":true,"sysDnsServerUpdate":true,"sysDnsServerDestroy":true,"sysDnsServerQuery":true,"sysDomainGet":true,"sysDomainCreate":true,"sysDomainUpdate":true,"sysDomainDestroy":true,"sysDomainQuery":true,"sysMailingListGet":true,"sysMailingListCreate":true,"sysMailingListUpdate":true,"sysMailingListDestroy":true,"sysMailingListQuery":true,"sysOAuthClientGet":true,"sysOAuthClientCreate":true,"sysOAuthClientUpdate":true,"sysOAuthClientDestroy":true,"sysOAuthClientQuery":true,"sysQueuedMessageGet":true,"sysQueuedMessageCreate":true,"sysQueuedMessageUpdate":true,"sysQueuedMessageDestroy":true,"sysQueuedMessageQuery":true,"sysRoleGet":true,"sysRoleCreate":true,"sysRoleUpdate":true,"sysRoleDestroy":true,"sysRoleQuery":true},"description":"Tenant Administrator","roleIds":{}},"role-c":{"disabledPermissions":{},"memberTenantId":null,"enabledPermissions":{"emailSend":true,"emailReceive":true,"calendarAlarmsSend":true,"calendarSchedulingSend":true,"calendarSchedulingReceive":true,"jmapPushSubscriptionGet":true,"jmapPushSubscriptionCreate":true,"jmapPushSubscriptionUpdate":true,"jmapPushSubscriptionDestroy":true,"jmapMailboxGet":true,"jmapMailboxChanges":true,"jmapMailboxQuery":true,"jmapMailboxQueryChanges":true,"jmapMailboxCreate":true,"jmapMailboxUpdate":true,"jmapMailboxDestroy":true,"jmapThreadGet":true,"jmapThreadChanges":true,"jmapEmailGet":true,"jmapEmailChanges":true,"jmapEmailQuery":true,"jmapEmailQueryChanges":true,"jmapEmailCreate":true,"jmapEmailUpdate":true,"jmapEmailDestroy":true,"jmapEmailCopy":true,"jmapEmailImport":true,"jmapEmailParse":true,"jmapSearchSnippetGet":true,"jmapIdentityGet":true,"jmapIdentityChanges":true,"jmapIdentityCreate":true,"jmapIdentityUpdate":true,"jmapIdentityDestroy":true,"jmapEmailSubmissionGet":true,"jmapEmailSubmissionChanges":true,"jmapEmailSubmissionQuery":true,"jmapEmailSubmissionQueryChanges":true,"jmapEmailSubmissionCreate":true,"jmapEmailSubmissionUpdate":true,"jmapEmailSubmissionDestroy":true,"jmapVacationResponseGet":true,"jmapVacationResponseCreate":true,"jmapVacationResponseUpdate":true,"jmapVacationResponseDestroy":true,"jmapSieveScriptGet":true,"jmapSieveScriptQuery":true,"jmapSieveScriptValidate":true,"jmapSieveScriptCreate":true,"jmapSieveScriptUpdate":true,"jmapSieveScriptDestroy":true,"jmapPrincipalGet":true,"jmapPrincipalQuery":true,"jmapPrincipalChanges":true,"jmapPrincipalQueryChanges":true,"jmapPrincipalGetAvailability":true,"jmapPrincipalCreate":true,"jmapPrincipalUpdate":true,"jmapPrincipalDestroy":true,"jmapQuotaGet":true,"jmapQuotaChanges":true,"jmapQuotaQuery":true,"jmapQuotaQueryChanges":true,"jmapBlobGet":true,"jmapBlobCopy":true,"jmapBlobLookup":true,"jmapBlobUpload":true,"jmapAddressBookGet":true,"jmapAddressBookChanges":true,"jmapAddressBookCreate":true,"jmapAddressBookUpdate":true,"jmapAddressBookDestroy":true,"jmapContactCardGet":true,"jmapContactCardChanges":true,"jmapContactCardQuery":true,"jmapContactCardQueryChanges":true,"jmapContactCardCreate":true,"jmapContactCardUpdate":true,"jmapContactCardDestroy":true,"jmapContactCardCopy":true,"jmapContactCardParse":true,"jmapFileNodeGet":true,"jmapFileNodeChanges":true,"jmapFileNodeQuery":true,"jmapFileNodeQueryChanges":true,"jmapFileNodeCreate":true,"jmapFileNodeUpdate":true,"jmapFileNodeDestroy":true,"jmapShareNotificationGet":true,"jmapShareNotificationChanges":true,"jmapShareNotificationQuery":true,"jmapShareNotificationQueryChanges":true,"jmapShareNotificationCreate":true,"jmapShareNotificationUpdate":true,"jmapShareNotificationDestroy":true,"jmapCalendarGet":true,"jmapCalendarChanges":true,"jmapCalendarCreate":true,"jmapCalendarUpdate":true,"jmapCalendarDestroy":true,"jmapCalendarEventGet":true,"jmapCalendarEventChanges":true,"jmapCalendarEventQuery":true,"jmapCalendarEventQueryChanges":true,"jmapCalendarEventCreate":true,"jmapCalendarEventUpdate":true,"jmapCalendarEventDestroy":true,"jmapCalendarEventCopy":true,"jmapCalendarEventParse":true,"jmapCalendarEventNotificationGet":true,"jmapCalendarEventNotificationChanges":true,"jmapCalendarEventNotificationQuery":true,"jmapCalendarEventNotificationQueryChanges":true,"jmapCalendarEventNotificationCreate":true,"jmapCalendarEventNotificationUpdate":true,"jmapCalendarEventNotificationDestroy":true,"jmapParticipantIdentityGet":true,"jmapParticipantIdentityChanges":true,"jmapParticipantIdentityCreate":true,"jmapParticipantIdentityUpdate":true,"jmapParticipantIdentityDestroy":true,"jmapCoreEcho":true,"imapAuthenticate":true,"imapAclGet":true,"imapAclSet":true,"imapMyRights":true,"imapListRights":true,"imapAppend":true,"imapCapability":true,"imapId":true,"imapCopy":true,"imapMove":true,"imapCreate":true,"imapDelete":true,"imapEnable":true,"imapExpunge":true,"imapFetch":true,"imapIdle":true,"imapList":true,"imapLsub":true,"imapNamespace":true,"imapRename":true,"imapSearch":true,"imapSort":true,"imapSelect":true,"imapExamine":true,"imapStatus":true,"imapStore":true,"imapSubscribe":true,"imapThread":true,"pop3Authenticate":true,"pop3List":true,"pop3Uidl":true,"pop3Stat":true,"pop3Retr":true,"pop3Dele":true,"sieveAuthenticate":true,"sieveListScripts":true,"sieveSetActive":true,"sieveGetScript":true,"sievePutScript":true,"sieveDeleteScript":true,"sieveRenameScript":true,"sieveCheckScript":true,"sieveHaveSpace":true,"davSyncCollection":true,"davExpandProperty":true,"davPrincipalAcl":true,"davPrincipalList":true,"davPrincipalMatch":true,"davPrincipalSearch":true,"davPrincipalSearchPropSet":true,"davFilePropFind":true,"davFilePropPatch":true,"davFileGet":true,"davFileMkCol":true,"davFileDelete":true,"davFilePut":true,"davFileCopy":true,"davFileMove":true,"davFileLock":true,"davFileAcl":true,"davCardPropFind":true,"davCardPropPatch":true,"davCardGet":true,"davCardMkCol":true,"davCardDelete":true,"davCardPut":true,"davCardCopy":true,"davCardMove":true,"davCardLock":true,"davCardAcl":true,"davCardQuery":true,"davCardMultiGet":true,"davCalPropFind":true,"davCalPropPatch":true,"davCalGet":true,"davCalMkCol":true,"davCalDelete":true,"davCalPut":true,"davCalCopy":true,"davCalMove":true,"davCalLock":true,"davCalAcl":true,"davCalQuery":true,"davCalMultiGet":true,"davCalFreeBusyQuery":true,"sysAccountSettingsGet":true,"sysAccountSettingsUpdate":true,"sysArchivedItemGet":true,"sysArchivedItemCreate":true,"sysArchivedItemUpdate":true,"sysArchivedItemDestroy":true,"sysArchivedItemQuery":true,"sysMaskedEmailGet":true,"sysMaskedEmailCreate":true,"sysMaskedEmailUpdate":true,"sysMaskedEmailDestroy":true,"sysMaskedEmailQuery":true,"sysPublicKeyGet":true,"sysPublicKeyCreate":true,"sysPublicKeyUpdate":true,"sysPublicKeyDestroy":true,"sysPublicKeyQuery":true,"sysSpamTrainingSampleGet":true,"sysSpamTrainingSampleUpdate":true,"sysSpamTrainingSampleDestroy":true,"sysSpamTrainingSampleQuery":true,"jmapFileNodeCopy":true},"description":"Group","roleIds":{}},"role-b":{"disabledPermissions":{},"memberTenantId":null,"enabledPermissions":{"authenticate":true,"authenticateWithAlias":true,"interactAi":true,"emailSend":true,"emailReceive":true,"calendarAlarmsSend":true,"calendarSchedulingSend":true,"calendarSchedulingReceive":true,"jmapPushSubscriptionGet":true,"jmapPushSubscriptionCreate":true,"jmapPushSubscriptionUpdate":true,"jmapPushSubscriptionDestroy":true,"jmapMailboxGet":true,"jmapMailboxChanges":true,"jmapMailboxQuery":true,"jmapMailboxQueryChanges":true,"jmapMailboxCreate":true,"jmapMailboxUpdate":true,"jmapMailboxDestroy":true,"jmapThreadGet":true,"jmapThreadChanges":true,"jmapEmailGet":true,"jmapEmailChanges":true,"jmapEmailQuery":true,"jmapEmailQueryChanges":true,"jmapEmailCreate":true,"jmapEmailUpdate":true,"jmapEmailDestroy":true,"jmapEmailCopy":true,"jmapEmailImport":true,"jmapEmailParse":true,"jmapSearchSnippetGet":true,"jmapIdentityGet":true,"jmapIdentityChanges":true,"jmapIdentityCreate":true,"jmapIdentityUpdate":true,"jmapIdentityDestroy":true,"jmapEmailSubmissionGet":true,"jmapEmailSubmissionChanges":true,"jmapEmailSubmissionQuery":true,"jmapEmailSubmissionQueryChanges":true,"jmapEmailSubmissionCreate":true,"jmapEmailSubmissionUpdate":true,"jmapEmailSubmissionDestroy":true,"jmapVacationResponseGet":true,"jmapVacationResponseCreate":true,"jmapVacationResponseUpdate":true,"jmapVacationResponseDestroy":true,"jmapSieveScriptGet":true,"jmapSieveScriptQuery":true,"jmapSieveScriptValidate":true,"jmapSieveScriptCreate":true,"jmapSieveScriptUpdate":true,"jmapSieveScriptDestroy":true,"jmapPrincipalGet":true,"jmapPrincipalQuery":true,"jmapPrincipalChanges":true,"jmapPrincipalQueryChanges":true,"jmapPrincipalGetAvailability":true,"jmapPrincipalCreate":true,"jmapPrincipalUpdate":true,"jmapPrincipalDestroy":true,"jmapQuotaGet":true,"jmapQuotaChanges":true,"jmapQuotaQuery":true,"jmapQuotaQueryChanges":true,"jmapBlobGet":true,"jmapBlobCopy":true,"jmapBlobLookup":true,"jmapBlobUpload":true,"jmapAddressBookGet":true,"jmapAddressBookChanges":true,"jmapAddressBookCreate":true,"jmapAddressBookUpdate":true,"jmapAddressBookDestroy":true,"jmapContactCardGet":true,"jmapContactCardChanges":true,"jmapContactCardQuery":true,"jmapContactCardQueryChanges":true,"jmapContactCardCreate":true,"jmapContactCardUpdate":true,"jmapContactCardDestroy":true,"jmapContactCardCopy":true,"jmapContactCardParse":true,"jmapFileNodeGet":true,"jmapFileNodeChanges":true,"jmapFileNodeQuery":true,"jmapFileNodeQueryChanges":true,"jmapFileNodeCreate":true,"jmapFileNodeUpdate":true,"jmapFileNodeDestroy":true,"jmapShareNotificationGet":true,"jmapShareNotificationChanges":true,"jmapShareNotificationQuery":true,"jmapShareNotificationQueryChanges":true,"jmapShareNotificationCreate":true,"jmapShareNotificationUpdate":true,"jmapShareNotificationDestroy":true,"jmapCalendarGet":true,"jmapCalendarChanges":true,"jmapCalendarCreate":true,"jmapCalendarUpdate":true,"jmapCalendarDestroy":true,"jmapCalendarEventGet":true,"jmapCalendarEventChanges":true,"jmapCalendarEventQuery":true,"jmapCalendarEventQueryChanges":true,"jmapCalendarEventCreate":true,"jmapCalendarEventUpdate":true,"jmapCalendarEventDestroy":true,"jmapCalendarEventCopy":true,"jmapCalendarEventParse":true,"jmapCalendarEventNotificationGet":true,"jmapCalendarEventNotificationChanges":true,"jmapCalendarEventNotificationQuery":true,"jmapCalendarEventNotificationQueryChanges":true,"jmapCalendarEventNotificationCreate":true,"jmapCalendarEventNotificationUpdate":true,"jmapCalendarEventNotificationDestroy":true,"jmapParticipantIdentityGet":true,"jmapParticipantIdentityChanges":true,"jmapParticipantIdentityCreate":true,"jmapParticipantIdentityUpdate":true,"jmapParticipantIdentityDestroy":true,"jmapCoreEcho":true,"imapAuthenticate":true,"imapAclGet":true,"imapAclSet":true,"imapMyRights":true,"imapListRights":true,"imapAppend":true,"imapCapability":true,"imapId":true,"imapCopy":true,"imapMove":true,"imapCreate":true,"imapDelete":true,"imapEnable":true,"imapExpunge":true,"imapFetch":true,"imapIdle":true,"imapList":true,"imapLsub":true,"imapNamespace":true,"imapRename":true,"imapSearch":true,"imapSort":true,"imapSelect":true,"imapExamine":true,"imapStatus":true,"imapStore":true,"imapSubscribe":true,"imapThread":true,"pop3Authenticate":true,"pop3List":true,"pop3Uidl":true,"pop3Stat":true,"pop3Retr":true,"pop3Dele":true,"sieveAuthenticate":true,"sieveListScripts":true,"sieveSetActive":true,"sieveGetScript":true,"sievePutScript":true,"sieveDeleteScript":true,"sieveRenameScript":true,"sieveCheckScript":true,"sieveHaveSpace":true,"davSyncCollection":true,"davExpandProperty":true,"davPrincipalAcl":true,"davPrincipalList":true,"davPrincipalMatch":true,"davPrincipalSearch":true,"davPrincipalSearchPropSet":true,"davFilePropFind":true,"davFilePropPatch":true,"davFileGet":true,"davFileMkCol":true,"davFileDelete":true,"davFilePut":true,"davFileCopy":true,"davFileMove":true,"davFileLock":true,"davFileAcl":true,"davCardPropFind":true,"davCardPropPatch":true,"davCardGet":true,"davCardMkCol":true,"davCardDelete":true,"davCardPut":true,"davCardCopy":true,"davCardMove":true,"davCardLock":true,"davCardAcl":true,"davCardQuery":true,"davCardMultiGet":true,"davCalPropFind":true,"davCalPropPatch":true,"davCalGet":true,"davCalMkCol":true,"davCalDelete":true,"davCalPut":true,"davCalCopy":true,"davCalMove":true,"davCalLock":true,"davCalAcl":true,"davCalQuery":true,"davCalMultiGet":true,"davCalFreeBusyQuery":true,"sysAccountPasswordGet":true,"sysAccountPasswordUpdate":true,"sysAccountSettingsGet":true,"sysAccountSettingsUpdate":true,"sysApiKeyGet":true,"sysApiKeyCreate":true,"sysApiKeyUpdate":true,"sysApiKeyDestroy":true,"sysApiKeyQuery":true,"sysAppPasswordGet":true,"sysAppPasswordCreate":true,"sysAppPasswordUpdate":true,"sysAppPasswordDestroy":true,"sysAppPasswordQuery":true,"sysArchivedItemGet":true,"sysArchivedItemCreate":true,"sysArchivedItemUpdate":true,"sysArchivedItemDestroy":true,"sysArchivedItemQuery":true,"sysMaskedEmailGet":true,"sysMaskedEmailCreate":true,"sysMaskedEmailUpdate":true,"sysMaskedEmailDestroy":true,"sysMaskedEmailQuery":true,"sysPublicKeyGet":true,"sysPublicKeyCreate":true,"sysPublicKeyUpdate":true,"sysPublicKeyDestroy":true,"sysPublicKeyQuery":true,"sysSpamTrainingSampleGet":true,"sysSpamTrainingSampleUpdate":true,"sysSpamTrainingSampleDestroy":true,"sysSpamTrainingSampleQuery":true,"jmapFileNodeCopy":true},"description":"User","roleIds":{}}}}`,
		`{
		  "@type": "create",
		  "object": "Domain",
		  "value": {
		    "domain-b": {
		      "aliases": {},
		      "catchAllAddress": null,
		      "certificateManagement": {
		        "@type": "Manual"
		      },
		      "dkimManagement": {
		        "@type": "Manual"
		      },
		      "subAddressing": {
		        "@type": "Enabled"
		      },
		      "logo": null,
		      "isEnabled": true,
		      "memberTenantId": null,
		      "description": null,
		      "name": "<<.domain>>",
		      "directoryId": null,
		      "allowRelaying": false,
		      "dnsManagement": {
		        "@type": "Manual"
		      },
		      "reportAddressUri": "mailto:postmaster"
		    }
		  }
		}`,
		`{
		  "@type": "create",
		  "object": "Account",
		  "value": {
		    "account-d": {
		      "@type": "User",
		      "name": "<<.masterusername>>",
		      "memberGroupIds": {},
		      "locale": "en_US",
		      "quotas": {},
		      "aliases": {},
		      "domainId": "#domain-b",
		      "memberTenantId": null,
		      "permissions": {
		        "@type": "Merge",
		        "enabledPermissions": {
		          "impersonate": true
		        },
		        "disabledPermissions": {}
		      },
		      "encryptionAtRest": {
		        "@type": "Disabled"
		      },
		      "credentials": {
		        "0": {
		          "@type": "Password",
		          "expiresAt": null,
		          "secret": "<<.masterpassword>>",
		          "allowedIps": {}
		        }
		      },
		      "description": "Master",
		      "timeZone": null,
		      "roles": {
		        "@type": "User"
		      }
		    },
		    "account-b": {
		      "@type": "User",
		      "name": "<<.adminusername>>",
		      "memberGroupIds": {},
		      "locale": "en_US",
		      "quotas": {},
		      "aliases": {},
		      "domainId": "#domain-b",
		      "memberTenantId": null,
		      "permissions": {
		        "@type": "Inherit"
		      },
		      "encryptionAtRest": {
		        "@type": "Disabled"
		      },
		      "credentials": {
		        "0": {
		          "@type": "Password",
		          "expiresAt": null,
		          "secret": "<<.adminpassword>>",
		          "allowedIps": {}
		        }
		      },
		      "description": "System administrator",
		      "timeZone": null,
		      "roles": {
		        "@type": "Admin"
		      }
		    }
		  }
		}`,
		`{"@type":"update","object":"Authentication","value":{"defaultGroupRoleIds":{"#role-c":true},"defaultAdminRoleIds":{"#role-e":true,"#role-b":true},"defaultUserRoleIds":{"#role-b":true},"maxAppPasswords":5,"defaultTenantRoleIds":{"#role-d":true,"#role-b":true},"directoryId":null,"passwordMinLength":1,"passwordDefaultExpiry":null,"passwordHashAlgorithm":"argon2id","passwordMinStrength":"zero","passwordMaxLength":128,"maxApiKeys":5}}`,
		`{"@type":"update","object":"Sharing","value":{"allowDirectoryQueries":<<.dirquery>>,"maxShares":10}}`,
		`{
		  "@type": "update",
		  "object": "SystemSettings",
		  "value": {
		    "defaultHostname": "<<.hostname>>",
		    "threadPoolSize": null,
		    "defaultCertificateId": null,
		    "proxyTrustedNetworks": {},
		    "defaultDomainId": "#domain-b",
		    "services": {
		      "caldav": {
		        "hostname": null,
		        "cleartext": false
		      },
		      "carddav": {
		        "hostname": null,
		        "cleartext": false
		      },
		      "imap": {
		        "hostname": null,
		        "cleartext": false
		      },
		      "jmap": {
		        "hostname": null,
		        "cleartext": false
		      },
		      "managesieve": {
		        "hostname": null,
		        "cleartext": false
		      },
		      "pop3": {
		        "hostname": null,
		        "cleartext": false
		      },
		      "smtp": {
		        "hostname": null,
		        "cleartext": false
		      },
		      "webdav": {
		        "hostname": null,
		        "cleartext": false
		      }
		    },
		    "mailExchangers": {
		      "0": {
		        "priority": 10,
		        "hostname": null
		      }
		    },
		    "maxConnections": 8192,
		    "providerInfo": {}
		  }
		}`,
		`{"@type":"update","object":"DataRetention","value":{"archiveDeletedAccountsFor":null,"blobCleanupSchedule":{"@type":"Daily","hour":4,"minute":0},"expungeSchedule":{"@type":"Daily","hour":0,"minute":0},"dataCleanupSchedule":{"@type":"Daily","hour":2,"minute":0},"expungeTrashAfter":2592000000,"holdMtaReportsFor":2592000000,"metricsCollectionInterval":{"@type":"Hourly","minute":0},"holdMetricsFor":7776000000,"holdTracesFor":2592000000,"expungeShareNotifyAfter":2592000000,"expungeSchedulingInboxAfter":2592000000,"maxChangesHistory":10000,"expungeSubmissionsAfter":259200000,"archiveDeletedItemsFor":null}}`,
		`{"@type":"update","object":"BlobStore","value":{"@type":"Default"}}`,
		`{"@type":"update","object":"InMemoryStore","value":{"@type":"Default"}}`,
		`{"@type":"update","object":"SearchStore","value":{"@type":"Default"}}`,
	}
)

func skip(t *testing.T) bool {
	if os.Getenv("CI") == "woodpecker" {
		t.Skip("Skipping tests because CI==woodpecker")
		return true
	}
	if os.Getenv("CI_SYSTEM_NAME") == "woodpecker" {
		t.Skip("Skipping tests because CI_SYSTEM_NAME==woodpecker")
		return true
	}
	if os.Getenv("USE_TESTCONTAINERS") == "false" {
		t.Skip("Skipping tests because USE_TESTCONTAINERS==false")
		return true
	}
	return false
}

type StalwartTest struct {
	t           *testing.T
	ip          string
	imapPort    uint16
	container   *testcontainers.DockerContainer
	ctx         context.Context
	cancelCtx   context.CancelFunc
	client      *Client
	logger      *oclog.Logger
	jmapBaseUrl *url.URL
	sessionUrl  *url.URL

	io.Closer
}

func (s *StalwartTest) Close() error {
	if s.container != nil {
		var c testcontainers.Container = s.container
		testcontainers.CleanupContainer(s.t, c)
	}
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
	return nil
}

func (s *StalwartTest) Context(session *Session) Context {
	return Context{
		Session:        session,
		Context:        s.ctx,
		Logger:         s.logger,
		AcceptLanguage: "",
	}
}

func (s *StalwartTest) Session(username string) *Session {
	session, jerr := s.client.FetchSession(s.ctx, s.sessionUrl, username, s.logger)
	require.NoError(s.t, jerr)
	require.NotNil(s.t, session.Capabilities.Mail)
	require.NotNil(s.t, session.Capabilities.Calendars)
	require.NotNil(s.t, session.Capabilities.Contacts)

	// we have to overwrite the hostname in JMAP URL because the container
	// will know its name to be a random Docker container identifier, or
	// "localhost" as we defined the hostname in the Stalwart configuration,
	// and we also need to overwrite the port number as its not mapped
	session.JmapUrl.Host = s.jmapBaseUrl.Host
	session.JmapUrl.Scheme = "http" // replace https with http
	session.WebsocketUrl.Host = s.jmapBaseUrl.Host
	session.WebsocketUrl.Scheme = "ws" // replace wss with ws

	if v, err := replaceHostProto(session.ApiUrl, "http", s.jmapBaseUrl.Host); err != nil {
		require.NoError(s.t, err)
	} else {
		session.ApiUrl = v
	}
	if v, err := replaceHostProto(session.DownloadUrl, "http", s.jmapBaseUrl.Host); err != nil {
		require.NoError(s.t, err)
	} else {
		session.DownloadUrl = v
	}
	if v, err := replaceHostProto(session.UploadUrl, "http", s.jmapBaseUrl.Host); err != nil {
		require.NoError(s.t, err)
	} else {
		session.UploadUrl = v
	}
	if v, err := replaceHostProto(session.EventSourceUrl, "http", s.jmapBaseUrl.Host); err != nil {
		require.NoError(s.t, err)
	} else {
		session.EventSourceUrl = v
	}

	return &session
}

type cliLogConsumer struct {
	lines *[]string
}

func (c *cliLogConsumer) Accept(l testcontainers.Log) {
	s := string(l.Content)
	log.Printf("CLI: %s", s)
	*c.lines = append(*c.lines, s)
}

type printingLogConsumer struct {
	prefix string
}

// 2026/06/05 16:18:36 STALWART: 2026-06-05T14:18:36Z INFO Shutting down Stalwart Server (server.shutdown) causedBy = "SIGTERM"
var printingLogConsumerRegex = regexp.MustCompile(`^(\d\d\d\d-\d\d-\d\dT\d\d:\d\d:\d\dZ)\s+(\S+)\s+(.+)$`)

func (c *printingLogConsumer) Accept(l testcontainers.Log) {
	str := string(l.Content)
	if m := printingLogConsumerRegex.FindAllStringSubmatch(str, 3); m != nil {
		log.Printf("\x1b[32;1m%s\x1b[0m \x1b[34m%s\x1b[0m \x1b[31;1m%s\x1b[0m %s", c.prefix, m[0][1], m[0][2], m[0][3])
	} else {
		log.Printf("%s: %s", c.prefix, str)
	}
}

func withDirectoryQueries(allowDirectoryQueries bool) func(map[string]any) {
	return func(m map[string]any) {
		m["dirquery"] = strconv.FormatBool(allowDirectoryQueries)
	}
}

func applySnapshot(ctx context.Context, net *testcontainers.DockerNetwork, uri, user, password string, content *strings.Reader) ([]string, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name: "Dockerfile",
		Mode: 0600,
		Size: int64(len(cliDockerfile)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write([]byte(cliDockerfile)); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}

	output := []string{}

	cliArch := "x86_64"
	switch runtime.GOARCH {
	case "amd64":
		cliArch = "x86_64"
	case "arm64", "arm64be":
		cliArch = "aarch64"
	case "arm", "armbe":
		cliArch = "armv7"
	default:
		return nil, fmt.Errorf("unsupported architecture: '%s'", runtime.GOARCH)
	}

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithDockerfile(testcontainers.FromDockerfile{
			ContextArchive: bytes.NewReader(buf.Bytes()),
			KeepImage:      true, // speeds up subsequent test runs by using the Docker cache
			BuildArgs: map[string]*string{
				"ARCH": &cliArch,
			},
		}),
		testcontainers.WithCmdArgs("/usr/local/bin/stalwart-cli", "apply", "--json", "--no-color", "--file", "/snapshot.json"),
		testcontainers.WithEnv(map[string]string{
			"STALWART_URL":      uri,
			"STALWART_USER":     user,
			"STALWART_PASSWORD": password,
		}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			Reader:            content,
			ContainerFilePath: "/snapshot.json",
			FileMode:          0o700,
		}),
		testcontainers.WithLogConsumers(&cliLogConsumer{lines: &output}),
		testcontainers.WithWaitStrategy(wait.ForExit()),
	}

	if net != nil {
		opts = append(opts, network.WithNetwork([]string{"cli"}, net))
	}

	container, err := testcontainers.Run(ctx, "", opts...)
	if err != nil {
		return nil, err
	}

	rc := 0
	s, stateErr := container.State(ctx)
	if stateErr == nil {
		rc = s.ExitCode
	}

	if err := container.Terminate(ctx); err != nil {
		return nil, err
	}

	if stateErr != nil {
		return nil, stateErr
	}
	if rc != 0 {
		return nil, fmt.Errorf("cli returned exit code %d", rc)
	}
	return output, nil
}

func postJmap(ctx context.Context, h http.Client, url string, username string, password string, body map[string]any) (map[string]any, error) {
	bb, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bb))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)
	resp, err := h.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.Status != "200 OK" {
		return nil, fmt.Errorf("failed POST onto the JMAP API endpoint at %s: %s", url, resp.Status)
	}

	result := map[string]any{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func createManagementObject(ctx context.Context, h http.Client, url string, adminUsername string, adminPassword string, objectType string, obj map[string]any) (string, error) {
	result, err := postJmap(ctx, h, url, adminUsername, adminPassword, map[string]any{
		"using": []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": []any{
			[]any{
				fmt.Sprintf("x:%s/set", objectType),
				map[string]any{
					"create": map[string]any{
						"n": obj,
					},
				},
				"c",
			},
		},
	})
	if err != nil {
		return "", err
	}

	text := ""
	{
		if b, err := json.MarshalIndent(result, "", "  "); err == nil {
			text = string(b)
		}
	}

	if methodResponses, ok := result["methodResponses"]; !ok {
		return "", fmt.Errorf("response has no methodResponses: %s", text)
	} else {
		methodResponses := methodResponses.([]any)
		if len(methodResponses) != 1 {
			return "", fmt.Errorf("response has a methodResponses that doesn't have a single item: %s", text)

		}
		firstMethodResponse := methodResponses[0].([]any)
		if len(firstMethodResponse) != 3 {
			return "", fmt.Errorf("response has a methodResponses with one item that is a JMAP response that does not have 3 elements: %s", text)
		}
		payload := firstMethodResponse[1]
		if payload == nil {
			return "", fmt.Errorf("response has a first methodResponse without any payload: %s", text)
		}
		if created, ok := (payload.(map[string]any))["created"]; !ok {
			return "", fmt.Errorf("response has a first methodResponse with a payload that has no 'created': %s", text)
		} else {
			created := created.(map[string]any)
			if specifically, ok := created["n"]; !ok {
				return "", fmt.Errorf("response has a first methodResponse with a payload that has no 'n': %s", text)
			} else {
				specifically := specifically.(map[string]any)
				if id, ok := specifically["id"]; !ok {
					return "", fmt.Errorf("response has a first methodResponse with a payload with a created with an 'n' that has no 'id': %s", text)
				} else {
					return id.(string), nil
				}
			}
		}
	}
}

func createJmapClient(container *testcontainers.DockerContainer, ctx context.Context, auth HttpJmapClientAuthenticator, logger *oclog.Logger) (Client, *url.URL, *url.URL, error) {
	ip, err := container.Host(ctx)
	if err != nil {
		return Client{}, nil, nil, err
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = time.Duration(30 * time.Second)
	tr.TLSClientConfig = tlsConfig
	jh := *http.DefaultClient
	jh.Transport = tr

	wsd := &websocket.Dialer{
		TLSClientConfig:  tlsConfig,
		HandshakeTimeout: time.Duration(10) * time.Second,
	}

	jmapPort, err := container.MappedPort(ctx, httpPort)
	if err != nil {
		return Client{}, nil, nil, err
	}
	jmapBaseUrl := &url.URL{
		Scheme: "http",
		Host:   ip + ":" + jmapPort.Port(),
		Path:   "/",
	}
	sessionUrl := jmapBaseUrl.JoinPath(".well-known", "jmap")

	if Wireshark != "" {
		fmt.Printf("\x1b[45;37;1m Starting Wireshark on port %v \x1b[0m\n", jmapPort)
		attr := os.ProcAttr{
			Dir:   ".",
			Env:   os.Environ(),
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		}
		cmd := []string{Wireshark, "-pkSl", "-i", "lo", "-f", fmt.Sprintf("port %d", jmapPort.Num()), "-Y", "http||websocket"}
		process, err := os.StartProcess(Wireshark, cmd, &attr)
		if err != nil {
			return Client{}, nil, nil, err
		}
		err = process.Release()
		if err != nil {
			return Client{}, nil, nil, err
		}

		time.Sleep(10 * time.Second)
	}

	eventListener := nullHttpJmapApiClientEventListener{}

	api := NewHttpJmapClient(&jh, auth, eventListener)

	wscf, err := NewHttpWsClientFactory(wsd, auth, logger, eventListener)
	if err != nil {
		return Client{}, nil, nil, err
	}

	return NewClient(api, api, api, wscf), jmapBaseUrl, sessionUrl, nil
}

type ContextPasswordAuthHttpJmapClientAuthenticator struct {
	key string
}

var _ HttpJmapClientAuthenticator = &ContextPasswordAuthHttpJmapClientAuthenticator{}

func (h *ContextPasswordAuthHttpJmapClientAuthenticator) Authenticate(ctx context.Context, username string, _ *oclog.Logger, req *http.Request) Error {
	password := ctx.Value(h.key).(string)
	req.SetBasicAuth(username, password)
	return nil
}

func (h *ContextPasswordAuthHttpJmapClientAuthenticator) AuthenticateWS(ctx context.Context, username string, _ *oclog.Logger, headers http.Header) Error {
	password := ctx.Value(h.key).(string)
	headers.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	return nil
}

const useNetwork = false

func newStalwartTest(t *testing.T, options ...func(map[string]any)) (*StalwartTest, error) { //NOSONAR
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	ctx := t.Context()
	cancel := func() {}
	var _ context.CancelFunc = cancel // ignore context leak warning: it is passed in the struct and called in Close()

	// A master user name different from "master" does not seem to work as of the current Stalwart version
	//masterUsernameSuffix, err := pw.Generate(4+rand.Intn(28), 2, 0, false, true)
	//require.NoError(err)
	masterUsername := "master" //"master_" + masterUsernameSuffix
	masterPassword, err := pw.Generate(10+rand.Intn(28), 2, 0, false, true)
	if err != nil {
		return nil, err
	}

	adminUsername := "admin"
	adminPassword, err := pw.Generate(10+rand.Intn(28), 2, 0, false, true)
	if err != nil {
		return nil, err
	}

	recoveryAlias := "recovery"
	containerAlias := "stalwart"
	volumeName := "stalwart-test-data-volume"
	{
		str, err := pw.Generate(32, 12, 0, true, true)
		if err != nil {
			return nil, err
		}
		volumeName += "-" + str
	}
	hostname := "127.0.0.1"

	settings := map[string]any{
		"hostname":        hostname,
		"masterusername":  masterUsername,
		"masterpassword":  masterPassword, //Hash,
		"adminusername":   adminUsername,
		"adminpassword":   adminPassword,
		"httpPort":        httpPort,
		"httpsPort":       10443,
		"popsPort":        10995,
		"imapsPort":       imapsPort,
		"submissionsPort": 10465,
		"smtpPort":        10025,
		"dirquery":        "false",
		"domain":          "example.org",
	}
	for _, option := range options {
		option(settings)
	}

	var net *testcontainers.DockerNetwork
	{
		if useNetwork {
			if n, err := network.New(ctx); err != nil {
				return nil, err
			} else {
				testcontainers.CleanupNetwork(t, n)
				net = n
			}
		} else {
			net = nil
		}
	}

	httpWait := wait.ForHTTP("/.well-known/jmap")
	httpWait.Port = dockernetwork.MustParsePort(httpPort)

	configBuf := bytes.NewBufferString("")
	template.Must(template.New("config").Delims("<<", ">>").Parse(jsonConfigTemplate)).Execute(configBuf, settings)
	config := configBuf.String()
	{
		var recovery *testcontainers.DockerContainer
		{
			opts := []testcontainers.ContainerCustomizer{
				testcontainers.WithLogConsumers(&printingLogConsumer{prefix: "RECOVERY"}),
				testcontainers.WithExposedPorts(httpPort + "/tcp"),
				testcontainers.WithEnv(map[string]string{
					"STALWART_RECOVERY_ADMIN": strings.Join([]string{adminUsername, adminPassword}, ":"),
					"STALWART_RECOVERY_MODE":  "1",
				}),
				testcontainers.WithFiles(testcontainers.ContainerFile{
					Reader:            strings.NewReader(config),
					ContainerFilePath: "/etc/stalwart/config.json",
					FileMode:          0o666,
				}),
				testcontainers.WithWaitStrategyAndDeadline(
					30*time.Second,
					wait.ForMappedPort(httpPort),
					httpWait,
				),
				testcontainers.WithName(recoveryAlias),
				testcontainers.WithMounts(
					testcontainers.ContainerMount{
						Source: testcontainers.GenericVolumeMountSource{
							Name: volumeName,
						},
						Target:   "/var/lib/stalwart",
						ReadOnly: false,
					},
				),
			}
			if net != nil {
				opts = append(opts, network.WithNetwork([]string{recoveryAlias}, net))
			}

			if c, err := testcontainers.Run(ctx, stalwartImage, opts...); err != nil {
				return nil, err
			} else {
				testcontainers.CleanupContainer(t, c)
				recovery = c
			}
		}

		// import config using the cli
		{
			uri := ""
			if net != nil {
				uri = (&url.URL{Scheme: "http", Host: recoveryAlias + ":" + httpPort, Path: "/"}).String()
			} else {
				if ir, err := recovery.Inspect(ctx); err != nil {
					return nil, err
				} else {
					id := ir.Config.Hostname
					for _, network := range ir.NetworkSettings.Networks {
						id = network.IPAddress.String()
					}
					uri = (&url.URL{Scheme: "http", Host: id + ":" + httpPort, Path: "/"}).String()
				}
			}

			snapshot := []string{}
			{
				t := template.New("snapshot").Delims("<<", ">>")
				for _, line := range dumpTemplate {
					buf := bytes.NewBufferString("")
					p, err := t.Parse(line)
					if err != nil {
						return nil, err
					}
					err = p.Execute(buf, settings)
					if err != nil {
						return nil, err
					}

					var m map[string]any
					err = json.Unmarshal(buf.Bytes(), &m)
					if err != nil {
						return nil, err
					}
					b, err := json.Marshal(m)
					if err != nil {
						return nil, err
					}
					snapshot = append(snapshot, strings.TrimSpace(string(b)))
				}
			}
			text := strings.Join(snapshot, "\n")

			t.Logf("Snapshot:\n%s", text)
			output, err := applySnapshot(ctx, net, uri, adminUsername, adminPassword, strings.NewReader(text))
			if err != nil {
				return nil, err
			}
			t.Logf("Output of applying configuration:\n%s", strings.Join(output, ""))
		}

		if err := recovery.Terminate(ctx, testcontainers.RemoveVolumes()); err != nil {
			return nil, err
		}
	}

	var container *testcontainers.DockerContainer
	{
		opts := []testcontainers.ContainerCustomizer{
			testcontainers.WithLogConsumers(&printingLogConsumer{prefix: "STALWART"}),
			testcontainers.WithExposedPorts(httpPort+"/tcp", imapsPort+"/tcp"),
			testcontainers.WithFiles(testcontainers.ContainerFile{
				Reader:            strings.NewReader(config),
				ContainerFilePath: "/etc/stalwart/config.json",
				FileMode:          0o666,
			}),
			testcontainers.WithWaitStrategyAndDeadline(
				30*time.Second,
				wait.ForMappedPort(httpPort),
				httpWait,
			),
			testcontainers.WithName(containerAlias),
			testcontainers.WithMounts(
				testcontainers.ContainerMount{
					Source: testcontainers.GenericVolumeMountSource{
						Name: volumeName,
					},
					Target:   "/var/lib/stalwart",
					ReadOnly: false,
				},
			),
		}

		if net != nil {
			opts = append(opts, network.WithNetwork([]string{containerAlias}, net))
		}

		if c, err := testcontainers.Run(ctx, stalwartImage, opts...); err != nil {
			return nil, err
		} else {
			testcontainers.CleanupContainer(t, c, testcontainers.RemoveVolumes(volumeName))
			container = c
		}
	}

	ip, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	imapPort, err := container.MappedPort(ctx, imapsPort)
	if err != nil {
		return nil, err
	}

	domains := structs.Uniq(structs.Map(users[:], func(u User) string {
		parts := strings.Split(u.email, "@")
		return parts[1]
	}))
	slices.Sort(domains)

	// provision some things using Stalwart's JMAP based Management API
	{
		var h http.Client
		{
			tr := http.DefaultTransport.(*http.Transport).Clone()
			tr.ResponseHeaderTimeout = time.Duration(30 * time.Second)
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			h = *http.DefaultClient
			h.Transport = tr
		}

		apiPort, err := container.MappedPort(ctx, httpPort)
		require.NoError(t, err)

		apiPath := ""
		{
			// fetch JMAP session first
			session := map[string]any{}
			{
				url := fmt.Sprintf("http://%s:%d/.well-known/jmap", ip, apiPort.Num())
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				require.NoError(t, err)
				req.SetBasicAuth(adminUsername, adminPassword)
				resp, err := h.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()
				require.Equal(t, "200 OK", resp.Status)
				err = json.NewDecoder(resp.Body).Decode(&session)
				require.NoError(t, err)
			}
			if u, ok := session["apiUrl"]; ok {
				p, err := url.Parse(u.(string))
				require.NoError(t, err)
				apiPath = strings.TrimLeft(p.Path, "/")
			} else {
				require.FailNow(t, "failed to find apiUrl in JMAP Session")
			}
		}
		url := fmt.Sprintf("http://%s:%d/%s", ip, apiPort.Num(), apiPath)

		domainIds := map[string]string{}
		for _, domain := range domains {
			fmt.Printf("Creating domain '%v'\n", domain)
			id, err := createManagementObject(ctx, h, url, adminUsername, adminPassword, "Domain", map[string]any{
				"name":                  domain,
				"aliases":               map[string]any{},
				"certificateManagement": map[string]any{"@type": "Manual"},
				"dkimManagement":        map[string]any{"@type": "Manual"},
				"dnsManagement":         map[string]any{"@type": "Manual"},
				"subAddressing":         map[string]any{"@type": "Enabled"},
			})
			require.NoError(t, err)
			domainIds[domain] = id
		}

		for _, user := range users {
			fmt.Printf("Creating individual '%v'\n", user.name)

			localPart := ""
			domain := ""
			domainId := ""
			{
				parts := strings.Split(user.alias, "@")
				localPart = parts[0]
				domain = parts[1]
				if v, ok := domainIds[domain]; ok {
					domainId = v
				} else {
					require.FailNow(t, "failed to find id for domain '%s' in user email address '%s'", domain, user.email)
				}
			}

			_, err := createManagementObject(ctx, h, url, adminUsername, adminPassword, "Account", map[string]any{
				"@type":       "User",
				"name":        user.name,
				"description": user.description,
				"aliases": map[string]any{
					"0": map[string]any{
						"enabled":  true,
						"name":     localPart,
						"domainId": domainId,
					},
				},
				"credentials": map[string]any{
					"0": map[string]any{
						"@type":      "Password",
						"secret":     user.password,
						"allowedIps": map[string]any{},
						"expiresAt":  nil,
					},
				},
				"domainId":         domainId,
				"encryptionAtRest": map[string]any{"@type": "Disabled"},
				"memberGroupIds":   map[string]any{},
				"permissions":      map[string]any{"@type": "Inherit"},
				"quotas": map[string]any{
					"maxEmails":    200000,
					"maxDiskQuota": 20000000000,
				},
				"roles": map[string]any{"@type": "User"},
			})
			require.NoError(t, err)
		}

		// fetch all users and double-check whether they are all in there, but also use this
		// to detect which primary email address has been assigned to the user by Stalwart,
		// and then store it into the user object
		{
			result, err := postJmap(ctx, h, url, adminUsername, adminPassword, map[string]any{
				"using": []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
				"methodCalls": []any{
					[]any{
						"x:Account/get",
						map[string]any{},
						"0",
					},
				},
			})
			require.NoError(t, err)

			usersByName := structs.Index(users[:], func(u User) string { return u.name })
			{
				mr := result["methodResponses"].([]any)
				f := mr[0].([]any)
				p := f[1].(map[string]any)
				l := p["list"].([]any)
				for _, u := range l {
					u := u.(map[string]any)
					name := u["name"].(string)
					email := u["emailAddress"].(string)
					if match, ok := usersByName[name]; ok {
						require.Equal(t, match.email, email)
						aliases := u["aliases"].(map[string]any)
						require.Len(t, aliases, 1)
						for _, alias := range aliases {
							parts := strings.Split(match.alias, "@")
							alias := alias.(map[string]any)
							require.True(t, alias["enabled"].(bool))
							require.Equal(t, parts[0], alias["name"].(string))
						}
						delete(usersByName, name)
					}
				}
			}
			require.Empty(t, usersByName)
			text := ""
			{
				if b, err := json.MarshalIndent(result, "", "  "); err == nil {
					text = string(b)
				}
			}
			t.Logf("Accounts:\n%s", text)
		}

		{

			loggerImpl := oclog.NewLogger(oclog.Level("trace"))
			logger := &loggerImpl

			auth := &ContextPasswordAuthHttpJmapClientAuthenticator{key: "password"}

			j, _, sessionUrl, err := createJmapClient(container, ctx, auth, logger)
			if err != nil {
				return nil, err
			}

			// check whether we can fetch a session for the provisioned users
			for _, user := range users {
				ctx := context.WithValue(ctx, "password", user.password)
				session, err := j.FetchSession(ctx, sessionUrl, user.email, oclog.From(logger.With().Str("username", user.email).Str("password", user.password)))
				require.NoError(t, err, "failed to retrieve JMAP session for newly created principal '%s'", user.email)
				require.Equal(t, user.email, session.Username)
			}
		}
	}

	loggerImpl := oclog.NewLogger(oclog.Level("trace"))
	logger := &loggerImpl

	auth := NewMasterAuthHttpJmapClientAuthenticator(masterUsername, masterPassword)

	j, jmapBaseUrl, sessionUrl, err := createJmapClient(container, ctx, auth, logger)
	if err != nil {
		return nil, err
	}

	return &StalwartTest{
		t:           t,
		ip:          ip,
		imapPort:    imapPort.Num(),
		container:   container,
		ctx:         ctx,
		cancelCtx:   cancel,
		client:      &j,
		logger:      logger,
		jmapBaseUrl: jmapBaseUrl,
		sessionUrl:  sessionUrl,
	}, nil
}

var urlHostRegex = regexp.MustCompile(`^(https?://)(.+?)/(.+)$`)

func replaceHostProto(u string, proto string, host string) (string, error) {
	if m := urlHostRegex.FindAllStringSubmatch(u, -1); m != nil {
		return fmt.Sprintf("%s://%s/%s", proto, host, m[0][3]), nil
	} else {
		return "", fmt.Errorf("'%v' does not match '%v'", u, urlHostRegex)
	}
}

func pickRandomlyFromMap[K comparable, V any](m map[K]V, min int, max int) map[K]V {
	if min < 0 || max < 0 {
		panic("min and max must be >= 0")
	}
	l := len(m)
	if min > l || max > l {
		panic(fmt.Sprintf("min and max must be <= %d", l))
	}
	n := min + rand.Intn(max-min+1)
	if n == l {
		return m
	}
	// let's use a deep copy so we can remove elements as we pick them
	c := make(map[K]V, l)
	maps.Copy(c, m)
	// r will hold the results
	r := make(map[K]V, n)
	for range n {
		pick := rand.Intn(len(c))
		j := 0
		for k, v := range m {
			if j == pick {
				delete(c, k)
				r[k] = v
				break
			}
			j++
		}
	}
	return r
}

var productName = "jmaptest"

type TestJmapClient struct {
	h        *http.Client
	username string
	password string
	session  *Session
	u        *url.URL
	trace    bool
	color    bool
}

func NewTestJmapClient(session *Session, username string, password string, trace bool, color bool) (*TestJmapClient, error) {
	httpTransport := http.DefaultTransport.(*http.Transport).Clone()
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	httpTransport.TLSClientConfig = tlsConfig
	h := http.DefaultClient
	h.Transport = httpTransport

	u, err := url.Parse(session.ApiUrl)
	if err != nil {
		return nil, err
	}

	return &TestJmapClient{
		h:        h,
		trace:    trace,
		color:    color,
		username: username,
		password: password,
		session:  session,
		u:        u,
	}, nil
}

func (j *TestJmapClient) Close() error {
	return nil
}

type uploadedBlob struct {
	BlobId string `json:"blobId"`
	Size   int    `json:"size"`
	Type   string `json:"type"`
	Sha512 string `json:"sha:512"`
}

func (j *TestJmapClient) uploadBlob(accountId AccountId, data []byte, mimetype string) (uploadedBlob, error) { //NOSONAR
	uploadUrl := strings.ReplaceAll(j.session.UploadUrl, "{accountId}", string(accountId))
	req, err := http.NewRequest(http.MethodPost, uploadUrl, bytes.NewReader(data))
	if err != nil {
		return uploadedBlob{}, err
	}
	req.Header.Add("Content-Type", mimetype)
	req.SetBasicAuth(j.username, j.password)
	res, err := j.h.Do(req)
	if err != nil {
		return uploadedBlob{}, err
	}
	defer res.Body.Close()
	var response []byte = nil
	if j.trace {
		if b, err := httputil.DumpResponse(res, false); err == nil {
			response, err = io.ReadAll(res.Body)
			if err != nil {
				return uploadedBlob{}, err
			}
			p := pretty.Pretty(response)
			if j.color {
				p = pretty.Color(p, nil)
			}
			log.Printf("<== %s%s\n", b, p)
		}
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return uploadedBlob{}, fmt.Errorf("blob uploading to '%v': status is %s", uploadUrl, res.Status)
	}
	if response == nil {
		response, err = io.ReadAll(res.Body)
		if err != nil {
			return uploadedBlob{}, err
		}
	}

	var result uploadedBlob
	err = json.Unmarshal(response, &result)
	if err != nil {
		return uploadedBlob{}, err
	}

	return result, nil
}

func (j *TestJmapClient) command(body map[string]any) ([]any, error) { //NOSONAR
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, j.u.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	if j.trace {
		if b, err := httputil.DumpRequestOut(req, false); err == nil {
			p := pretty.Pretty(payload)
			if j.color {
				p = pretty.Color(p, nil)
			}
			log.Printf("==> %s%s\n", b, p)
		}
	}

	req.SetBasicAuth(j.username, j.password)
	resp, err := j.h.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var response []byte = nil
	if j.trace {
		if b, err := httputil.DumpResponse(resp, false); err == nil {
			response, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			p := pretty.Pretty(response)
			if j.color {
				p = pretty.Color(p, nil)
			}
			log.Printf("<== %s%s\n", b, p)
		}
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("JMAP command HTTP response status is %s", resp.Status)
	}
	if response == nil {
		response, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	r := map[string]any{}
	err = json.Unmarshal(response, &r)
	if err != nil {
		return nil, err
	}

	return r["methodResponses"].([]any), nil
}

type Commander[T any] struct {
	j       *TestJmapClient
	closure func([]any) (T, error)
}

func newCommander[T any](j *TestJmapClient, closure func([]any) (T, error)) Commander[T] {
	return Commander[T]{j: j, closure: closure}
}

func (c Commander[T]) command(body map[string]any) (T, error) {
	var zero T
	methodResponses, err := c.j.command(body)
	if err != nil {
		return zero, err
	}
	return c.closure(methodResponses)
}

func (j *TestJmapClient) objectsById(accountId AccountId, objectType ObjectType) (map[string]map[string]any, error) {
	m := map[string]map[string]any{}
	{
		body := map[string]any{
			"using": structs.Map(objectType.Namespaces, func(n JmapNamespace) string { return string(n) }),
			"methodCalls": []any{
				[]any{
					objectType.Name + "/get",
					map[string]any{
						"accountId": accountId,
					},
					"0",
				},
			},
		}
		result, err := newCommander(j, func(methodResponses []any) ([]any, error) {
			z := methodResponses[0].([]any)
			f := z[1].(map[string]any)
			if list, ok := f["list"]; ok {
				return list.([]any), nil
			} else {
				return nil, fmt.Errorf("methodResponse[1] has no 'list' attribute: %v", f)
			}
		}).command(body)
		if err != nil {
			return nil, err
		}
		for _, a := range result {
			obj := a.(map[string]any)
			id := obj["id"].(string)
			m[id] = obj
		}
	}
	return m, nil
}

func createName(person *gofakeit.PersonInfo) jscontact.Name {
	name := jscontact.Name{
		Type: jscontact.NameType,
	}
	comps := make([]jscontact.NameComponent, 2)
	comps[0] = jscontact.NameComponent{
		Type:  jscontact.NameComponentType,
		Kind:  jscontact.NameComponentKindGiven,
		Value: person.FirstName,
	}
	comps[1] = jscontact.NameComponent{
		Type:  jscontact.NameComponentType,
		Kind:  jscontact.NameComponentKindSurname,
		Value: person.LastName,
	}
	name.Components = comps
	name.IsOrdered = true
	name.DefaultSeparator = " "
	full := fmt.Sprintf("%s %s", person.FirstName, person.LastName)
	name.Full = full
	return name
}

func createNickName(_ *gofakeit.PersonInfo) jscontact.Nickname {
	name := gofakeit.PetName()
	contexts := pickRandoms(jscontact.NicknameContextPrivate, jscontact.NicknameContextWork)
	return jscontact.Nickname{
		Type:     jscontact.NicknameType,
		Name:     name,
		Contexts: orNilMap(toBoolMap(contexts)),
	}
}

func createEmail(person *gofakeit.PersonInfo, pref int) jscontact.EmailAddress {
	email := person.Contact.Email
	contexts := pickRandoms1(jscontact.EmailAddressContextWork, jscontact.EmailAddressContextPrivate)
	label := strings.ToLower(person.FirstName)
	return jscontact.EmailAddress{
		Type:     jscontact.EmailAddressType,
		Address:  email,
		Contexts: orNilMap(toBoolMap(contexts)),
		Label:    label,
		Pref:     uint(pref),
	}
}

func createSecondaryEmail(email string, pref int) jscontact.EmailAddress {
	contexts := pickRandoms(jscontact.EmailAddressContextWork, jscontact.EmailAddressContextPrivate)
	return jscontact.EmailAddress{
		Type:     jscontact.EmailAddressType,
		Address:  email,
		Contexts: orNilMap(toBoolMap(contexts)),
		Pref:     uint(pref),
	}
}

var idFirstLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var idOtherLetters = append(idFirstLetters, []rune("0123456789")...)

func id() string {
	n := 4 + rand.Intn(12-4+1)
	b := make([]rune, n)
	b[0] = idFirstLetters[rand.Intn(len(idFirstLetters))]
	for i := 1; i < n; i++ {
		b[i] = idOtherLetters[rand.Intn(len(idOtherLetters))]
	}
	return string(b)
}

func toHtml(text string) string {
	return "<!DOCTYPE html>\n<html>\n<body>\n" + strings.Join(htmlJoin(paraSplitter.Split(text, -1)), "\n") + "</body>\n</html>"
}

func htmlJoin(parts []string) []string {
	var result []string
	for i := range parts {
		result = append(result, fmt.Sprintf("<p>%v</p>", parts[i]))
	}
	return result
}

var paraSplitter = regexp.MustCompile("[\r\n]+")

var timezones = []string{
	"America/Adak",
	"America/Anchorage",
	"America/Chicago",
	"America/Denver",
	"America/Detroit",
	"America/Indiana/Knox",
	"America/Kentucky/Louisville",
	"America/Los_Angeles",
	"America/New_York",
	"Europe/Brussels",
	"Europe/Berlin",
	"Europe/Paris",
}

// https://www.w3.org/TR/css-color-3/#html4
var basicColors = []string{
	"black",
	"silver",
	"gray",
	"white",
	"maroon",
	"red",
	"purple",
	"fuchsia",
	"green",
	"lime",
	"olive",
	"yellow",
	"navy",
	"blue",
	"teal",
	"aqua",
}

/*
// https://www.w3.org/TR/SVG11/types.html#ColorKeywords
var extendedColors = []string{
	"aliceblue",
	"antiquewhite",
	"aqua",
	"aquamarine",
	"azure",
	"beige",
	"bisque",
	"black",
	"blanchedalmond",
	"blue",
	"blueviolet",
	"brown",
	"burlywood",
	"cadetblue",
	"chartreuse",
	"chocolate",
	"coral",
	"cornflowerblue",
	"cornsilk",
	"crimson",
	"cyan",
	"darkblue",
	"darkcyan",
	"darkgoldenrod",
	"darkgray",
	"darkgreen",
	"darkgrey",
	"darkkhaki",
	"darkmagenta",
	"darkolivegreen",
	"darkorange",
	"darkorchid",
	"darkred",
	"darksalmon",
	"darkseagreen",
	"darkslateblue",
	"darkslategray",
	"darkslategrey",
	"darkturquoise",
	"darkviolet",
	"deeppink",
	"deepskyblue",
	"dimgray",
	"dimgrey",
	"dodgerblue",
	"firebrick",
	"floralwhite",
	"forestgreen",
	"fuchsia",
	"gainsboro",
	"ghostwhite",
	"gold",
	"goldenrod",
	"gray",
	"grey",
	"green",
	"greenyellow",
	"honeydew",
	"hotpink",
	"indianred",
	"indigo",
	"ivory",
	"khaki",
	"lavender",
	"lavenderblush",
	"lawngreen",
	"lemonchiffon",
	"lightblue",
	"lightcoral",
	"lightcyan",
	"lightgoldenrodyellow",
	"lightgray",
	"lightgreen",
	"lightgrey",
	"lightpink",
	"lightsalmon",
	"lightseagreen",
	"lightskyblue",
	"lightslategray",
	"lightslategrey",
	"lightsteelblue",
	"lightyellow",
	"lime",
	"limegreen",
	"linen",
	"magenta",
	"maroon",
	"mediumaquamarine",
	"mediumblue",
	"mediumorchid",
	"mediumpurple",
	"mediumseagreen",
	"mediumslateblue",
	"mediumspringgreen",
	"mediumturquoise",
	"mediumvioletred",
	"midnightblue",
	"mintcream",
	"mistyrose",
	"moccasin",
	"navajowhite",
	"navy",
	"oldlace",
	"olive",
	"olivedrab",
	"orange",
	"orangered",
	"orchid",
	"palegoldenrod",
	"palegreen",
	"paleturquoise",
	"palevioletred",
	"papayawhip",
	"peachpuff",
	"peru",
	"pink",
	"plum",
	"powderblue",
	"purple",
	"red",
	"rosybrown",
	"royalblue",
	"saddlebrown",
	"salmon",
	"sandybrown",
	"seagreen",
	"seashell",
	"sienna",
	"silver",
	"skyblue",
	"slateblue",
	"slategray",
	"slategrey",
	"snow",
	"springgreen",
	"steelblue",
	"tan",
	"teal",
	"thistle",
	"tomato",
	"turquoise",
	"violet",
	"wheat",
	"white",
	"whitesmoke",
	"yellow",
	"yellowgreen",
}
*/

func propmap[T any](enabled bool, min int, max int, cardProperty *map[string]T, generator func(int, string) (T, error)) error {
	if !enabled {
		return nil
	}
	n := min + rand.Intn(max-min+1)

	o := make(map[string]T, n)
	for i := range n {
		id := id()
		itemForCard, err := generator(i, id)
		if err != nil {
			return err
		}
		o[id] = itemForCard
	}
	if len(o) > 0 {
		*cardProperty = o
	}
	return nil
}

func externalImageUri() string {
	return fmt.Sprintf("https://picsum.photos/id/%d/%d/%d", 1+rand.Intn(200), 200, 300)
}

func orNilMap[K comparable, V any](m map[K]V) map[K]V {
	if len(m) < 1 {
		return nil
	} else {
		return m
	}
}

func toBoolMap[K comparable](s []K) map[K]bool {
	m := make(map[K]bool, len(s))
	for _, e := range s {
		m[e] = true
	}
	return m
}

func toBoolPtrMap[K comparable](s []K) map[K]*bool {
	m := make(map[K]*bool, len(s))
	for _, e := range s {
		m[e] = truep
	}
	return m
}

func toBoolMapS[K comparable](s ...K) map[K]bool {
	m := make(map[K]bool, len(s))
	for _, e := range s {
		m[e] = true
	}
	return m
}

func pickRandom[T any](s ...T) T {
	return s[rand.Intn(len(s))]
}

func pickUser() User {
	return users[rand.Intn(len(users))]
}

func pickRandoms[T any](s ...T) []T {
	return pickRandomN(rand.Intn(len(s)), s...)
}

func pickRandomN[T any](n int, s ...T) []T {
	if n == 0 {
		return []T{}
	}
	result := make([]T, n)
	o := make([]T, len(s))
	copy(o, s)
	for i := range n {
		p := rand.Intn(len(o))
		result[i] = slices.Delete(o, p, p)[0]
	}
	return result
}

func pickRandoms1[T any](s ...T) []T {
	n := 1 + rand.Intn(len(s)-1)
	result := make([]T, n)
	o := make([]T, len(s))
	copy(o, s)
	for i := range n {
		p := rand.Intn(len(o))
		result[i] = slices.Delete(o, p, p)[0]
	}
	return result
}

func pickLanguage() string {
	return pickRandom("en-US", "en-GB", "en-AU")
}

func pickLocale() string {
	return pickRandom("en", "fr", "de")
}

func allBoxesAreTicked[S any](t *testing.T, s S, exceptions ...string) {
	v := reflect.ValueOf(s)
	typ := v.Type()
	tname := typ.Name()
	for i := range v.NumField() {
		name := typ.Field(i).Name
		if slices.Contains(exceptions, name) {
			log.Printf("%s[🍒] %s\n", tname, name)
			continue
		}
		value := v.Field(i).Bool()
		if value {
			log.Printf("%s[✅] %s\n", tname, name)
		} else {
			log.Printf("%s[❌] %s\n", tname, name)
		}
		require.True(t, value, "should be true: %v", name)
	}
}

func deepEqual[T any](t *testing.T, expected, actual T) {
	diff := ""
	if EnableTypes {
		diff = cmp.Diff(expected, actual)
	} else {
		diff = cmp.Diff(expected, actual, cmp.FilterPath(func(p cmp.Path) bool {
			switch sf := p.Last().(type) {
			case cmp.StructField:
				return sf.String() == ".Type"
			}
			return false
		}, cmp.Ignore()))
	}
	require.Empty(t, diff)
}

func containerTest[OBJ Idable, RESP GetResponse[OBJ], BOXES any, CHANGE Change](t *testing.T, //NOSONAR
	acc func(session *Session) AccountId,
	obj func(RESP) []OBJ,
	id func(OBJ) string,
	get func(s *StalwartTest, accountId AccountId, ids []string, ctx Context) (Result[RESP], error),
	update func(s *StalwartTest, accountId AccountId, id string, change CHANGE, ctx Context) (Result[OBJ], error),
	destroy func(s *StalwartTest, accountId AccountId, ids []string, ctx Context) (Result[map[string]SetError], error),
	fill func(s *StalwartTest, t *testing.T, accountId AccountId, count uint, ctx Context, _ User, principalIds []PrincipalId) (BOXES, []OBJ, SessionState, State, error),
	change func(OBJ) CHANGE,
	checkChanged func(t *testing.T, orig OBJ, change CHANGE, changed OBJ),
) {
	require := require.New(t)

	s, err := newStalwartTest(t, withDirectoryQueries(true))
	require.NoError(err)
	defer s.Close()

	user := pickUser()
	session := s.Session(user.email)
	ctx := s.Context(session)

	accountId := acc(session)

	// we first need to retrieve the list of all the Principals in order to be able to use and test sharing
	principalIds := []PrincipalId{}
	{
		result, err := s.client.GetPrincipals(accountId, []PrincipalId{}, ctx)
		require.NoError(err)
		require.NotEmpty(result.Payload.List)
		principalIds = structs.Map(result.Payload.List, func(p Principal) PrincipalId { return PrincipalId(p.Id) })
	}

	ss := EmptySessionState
	as := EmptyState

	// we need to fetch the ID of the default object that automatically exists for each user, in order to exclude it
	// from the tests below
	preExistingIds := []string{}
	{
		result, err := get(s, accountId, []string{}, ctx)
		require.NoError(err)
		require.Empty(result.Payload.GetNotFound())
		objs := obj(result.Payload)
		preExistingIds = structs.Map(objs, id)
		ss = result.GetSessionState()
		as = result.GetState()
	}

	// we are going to create a random amount of objects
	num := uint(5 + rand.Intn(30))
	{
		var all []OBJ
		var boxes BOXES
		{
			b, a, sessionState, state, err := fill(s, t, accountId, num, ctx, user, principalIds)
			require.NoError(err)
			require.Len(a, int(num))
			ss = sessionState
			as = state
			boxes = b
			all = a
		}

		{
			// lets retrieve all the existing objects by passing an empty ID slice
			result, err := get(s, accountId, []string{}, ctx)
			require.NoError(err)
			require.Empty(result.Payload.GetNotFound())
			objs := obj(result.Payload)
			// lets skip the objects that already exist since we did not create those
			found := structs.Filter(objs, func(a OBJ) bool { return !slices.Contains(preExistingIds, id(a)) })
			require.Len(found, int(num))
			m := structs.Index(found, id)
			require.Len(m, int(num))
			require.Equal(result.GetSessionState(), ss)
			require.Equal(result.GetState(), as)

			for _, a := range all {
				i := id(a)
				require.Contains(m, i)
				found, ok := m[i]
				require.True(ok)
				require.Equal(a, found)
			}

			ss = result.GetSessionState()
			as = result.GetState()
		}

		// lets retrieve every object we created by its ID
		for _, a := range all {
			i := id(a)
			result, err := get(s, accountId, []string{i}, ctx)
			require.NoError(err)
			require.Empty(result.Payload.GetNotFound())
			objs := obj(result.Payload)
			require.Len(objs, 1)
			require.Equal(result.GetSessionState(), ss)
			require.Equal(result.GetState(), as)
			require.Equal(objs[0], a)
		}

		// let's retrieve them all by their IDs, but this time all at once
		{
			ids := structs.Map(all, id)
			result, err := get(s, accountId, ids, ctx)
			require.NoError(err)
			require.Empty(result.Payload.GetNotFound())
			objs := obj(result.Payload)
			require.Len(objs, len(all))
			require.Equal(result.GetSessionState(), ss)
			require.Equal(result.GetState(), as)
			allById := structs.Index(all, id)
			for _, r := range result.Payload.GetList() {
				a, ok := allById[r.GetId()]
				require.True(ok, "failed to find object that was retrieved in mass ID request in the list of objects that were created")
				require.Equal(a, r)
			}
		}

		// lets modify each object
		for _, a := range all {
			i := id(a)
			ch := change(a)
			result, err := update(s, accountId, i, ch, ctx)
			require.NoError(err)
			require.NotEqual(a, result.Payload)
			require.Equal(result.GetSessionState(), ss)
			require.NotEqual(result.GetState(), as)
			checkChanged(t, a, ch, result.Payload)
		}

		// now lets delete each object that we created, all at once
		ids := structs.Map(all, id)
		{
			result, err := destroy(s, accountId, ids, ctx)
			require.NoError(err)
			require.Empty(result.Payload)
			require.Equal(result.GetSessionState(), ss)
			require.NotEqual(result.GetState(), as)
		}

		allBoxesAreTicked(t, boxes)
	}
}
