Feature: service health check


  Scenario: check default services health
    When a user requests these URLs with "GET" and no authentication
      | endpoint                                | service            |
      | http://%base_url_hostname%:9197/healthz | activitylog        |
      | http://%base_url_hostname%:9165/healthz | app-provider       |
      | http://%base_url_hostname%:9243/healthz | app-registry       |
      | http://%base_url_hostname%:9147/healthz | auth-basic         |
      | http://%base_url_hostname%:9167/healthz | auth-machine       |
      | http://%base_url_hostname%:9198/healthz | auth-service       |
      | http://%base_url_hostname%:9260/healthz | clientlog          |
      | http://%base_url_hostname%:9270/healthz | eventhistory       |
      | http://%base_url_hostname%:9141/healthz | frontend           |
      | http://%base_url_hostname%:9143/healthz | gateway            |
      | http://%base_url_hostname%:9124/healthz | graph              |
      | http://%base_url_hostname%:9161/healthz | groups             |
      | http://%base_url_hostname%:9239/healthz | idm                |
      | http://%base_url_hostname%:9134/healthz | idp                |
      | http://%base_url_hostname%:9234/healthz | nats               |
      | http://%base_url_hostname%:9281/healthz | ocm                |
      | http://%base_url_hostname%:9114/healthz | ocs                |
      | http://%base_url_hostname%:9255/healthz | postprocessing     |
      | http://%base_url_hostname%:9205/healthz | proxy              |
      | http://%base_url_hostname%:9224/healthz | search             |
      | http://%base_url_hostname%:9194/healthz | settings           |
      | http://%base_url_hostname%:9151/healthz | sharing            |
      | http://%base_url_hostname%:9139/healthz | sse                |
      | http://%base_url_hostname%:9179/healthz | storage-publiclink |
      | http://%base_url_hostname%:9156/healthz | storage-shares     |
      | http://%base_url_hostname%:9217/healthz | storage-system     |
      | http://%base_url_hostname%:9159/healthz | storage-users      |
      | http://%base_url_hostname%:9189/healthz | thumbnails         |
      | http://%base_url_hostname%:9214/healthz | userlog            |
      | http://%base_url_hostname%:9145/healthz | users              |
      | http://%base_url_hostname%:9104/healthz | web                |
      | http://%base_url_hostname%:9119/healthz | webdav             |
      | http://%base_url_hostname%:9279/healthz | webfinger          |
    Then the HTTP status code of responses on all endpoints should be "200"

  @env-config
  Scenario: check extra services health
    Given the following configs have been set:
      | config                 | value                                           |
      | OC_ADD_RUN_SERVICES  | audit,auth-app,auth-bearer,policies,invitations |
      | AUDIT_DEBUG_ADDR       | 0.0.0.0:9229                                    |
      | AUTH_APP_DEBUG_ADDR    | 0.0.0.0:9245                                    |
      | POLICIES_DEBUG_ADDR    | 0.0.0.0:9129                                    |
      | INVITATIONS_DEBUG_ADDR | 0.0.0.0:9269                                    |
    When a user requests these URLs with "GET" and no authentication
      | endpoint                                | service     |
      | http://%base_url_hostname%:9229/healthz | audit       |
      | http://%base_url_hostname%:9245/healthz | auth-app    |
      | http://%base_url_hostname%:9269/healthz | invitations |
      | http://%base_url_hostname%:9129/healthz | policies    |
    Then the HTTP status code of responses on all endpoints should be "200"


  Scenario: check default services readiness
    When a user requests these URLs with "GET" and no authentication
      | endpoint                               | service            |
      | http://%base_url_hostname%:9197/readyz | activitylog        |
      | http://%base_url_hostname%:9165/readyz | app-provider       |
      | http://%base_url_hostname%:9243/readyz | app-registry       |
      | http://%base_url_hostname%:9147/readyz | auth-basic         |
      | http://%base_url_hostname%:9167/readyz | auth-machine       |
      | http://%base_url_hostname%:9198/readyz | auth-service       |
      | http://%base_url_hostname%:9260/readyz | clientlog          |
      | http://%base_url_hostname%:9270/readyz | eventhistory       |
      | http://%base_url_hostname%:9141/readyz | frontend           |
      | http://%base_url_hostname%:9143/readyz | gateway            |
      | http://%base_url_hostname%:9161/readyz | groups             |
      | http://%base_url_hostname%:9239/readyz | idm                |
      | http://%base_url_hostname%:9234/readyz | nats               |
      | http://%base_url_hostname%:9281/readyz | ocm                |
      | http://%base_url_hostname%:9114/readyz | ocs                |
      | http://%base_url_hostname%:9255/readyz | postprocessing     |
      | http://%base_url_hostname%:9224/readyz | search             |
      | http://%base_url_hostname%:9194/readyz | settings           |
      | http://%base_url_hostname%:9151/readyz | sharing            |
      | http://%base_url_hostname%:9139/readyz | sse                |
      | http://%base_url_hostname%:9179/readyz | storage-publiclink |
      | http://%base_url_hostname%:9156/readyz | storage-shares     |
      | http://%base_url_hostname%:9217/readyz | storage-system     |
      | http://%base_url_hostname%:9159/readyz | storage-users      |
      | http://%base_url_hostname%:9189/readyz | thumbnails         |
      | http://%base_url_hostname%:9214/readyz | userlog            |
      | http://%base_url_hostname%:9145/readyz | users              |
      | http://%base_url_hostname%:9104/readyz | web                |
      | http://%base_url_hostname%:9119/readyz | webdav             |
      | http://%base_url_hostname%:9279/readyz | webfinger          |
    Then the HTTP status code of responses on all endpoints should be "200"

  @env-config
  Scenario: check extra services readiness
    Given the following configs have been set:
      | config                 | value                                           |
      | OC_ADD_RUN_SERVICES  | audit,auth-app,auth-bearer,policies,invitations |
      | AUDIT_DEBUG_ADDR       | 0.0.0.0:9229                                    |
      | AUTH_APP_DEBUG_ADDR    | 0.0.0.0:9245                                    |
      | AUTH_BEARER_DEBUG_ADDR | 0.0.0.0:9149                                    |
      | POLICIES_DEBUG_ADDR    | 0.0.0.0:9129                                    |
      | INVITATIONS_DEBUG_ADDR | 0.0.0.0:9269                                    |
    When a user requests these URLs with "GET" and no authentication
      | endpoint                               | service     |
      | http://%base_url_hostname%:9229/readyz | audit       |
      | http://%base_url_hostname%:9245/readyz | auth-app    |
      | http://%base_url_hostname%:9269/readyz | invitations |
      | http://%base_url_hostname%:9129/readyz | policies    |
    Then the HTTP status code of responses on all endpoints should be "200"

  @issue-10661
  Scenario: check default services readiness
    When a user requests these URLs with "GET" and no authentication
      | endpoint                               | service |
      | http://%base_url_hostname%:9124/readyz | graph   |
      | http://%base_url_hostname%:9134/readyz | idp     |
      | http://%base_url_hostname%:9205/readyz | proxy   |
    Then the HTTP status code of responses on all endpoints should be "200"

  @env-config @issue-10661
  Scenario: check extra services readiness
    Given the following configs have been set:
      | config                 | value        |
      | OC_ADD_RUN_SERVICES  | auth-bearer  |
      | AUTH_BEARER_DEBUG_ADDR | 0.0.0.0:9149 |
    When a user requests these URLs with "GET" and no authentication
      | endpoint                               | service     |
      | http://%base_url_hostname%:9149/readyz | auth-bearer |
    Then the HTTP status code of responses on all endpoints should be "200"
