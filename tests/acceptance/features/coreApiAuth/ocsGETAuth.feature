Feature: auth
  As a user
  I want to send GET request to various endpoints
  So that I can make sure the endpoints need proper authentication

  Background:
    Given user "Alice" has been created with default attributes

  @owncloud-issue-1337 @smokeTest
  Scenario: using OCS anonymously
    When a user requests these endpoints with "GET" and no authentication
      | endpoint                                                    |
      | /ocs/v1.php/apps/files_sharing/api/v1/remote_shares         |
      | /ocs/v2.php/apps/files_sharing/api/v1/remote_shares         |
      | /ocs/v1.php/apps/files_sharing/api/v1/remote_shares/pending |
      | /ocs/v2.php/apps/files_sharing/api/v1/remote_shares/pending |
      | /ocs/v1.php/apps/files_sharing/api/v1/shares                |
      | /ocs/v2.php/apps/files_sharing/api/v1/shares                |
      | /ocs/v1.php/cloud/apps                                      |
      | /ocs/v2.php/cloud/apps                                      |
      | /ocs/v1.php/cloud/groups                                    |
      | /ocs/v2.php/cloud/groups                                    |
      | /ocs/v1.php/cloud/users                                     |
      | /ocs/v2.php/cloud/users                                     |
      | /ocs/v1.php/privatedata/getattribute                        |
      | /ocs/v2.php/privatedata/getattribute                        |
    Then the HTTP status code of responses on all endpoints should be "401"

  @owncloud-issue-1338
  Scenario: ocs config end point accessible by unauthorized users
    When a user requests these endpoints with "GET" and no authentication
      | endpoint           |
      | /ocs/v1.php/config |
    Then the HTTP status code of responses on all endpoints should be "200"
    And the OCS status code of responses on all endpoints should be "100"
    When a user requests these endpoints with "GET" and no authentication
      | endpoint           |
      | /ocs/v2.php/config |
    Then the HTTP status code of responses on all endpoints should be "200"
    And the OCS status code of responses on all endpoints should be "200"

  @owncloud-issue-1337 @owncloud-issue-1336 @owncloud-issue-1335 @owncloud-issue-1334 @owncloud-issue-1333
  Scenario: using OCS with non-admin basic auth
    When the user "Alice" requests these endpoints with "GET" with basic auth
      | endpoint                                                    |
      | /ocs/v1.php/apps/files_sharing/api/v1/remote_shares         |
      | /ocs/v1.php/apps/files_sharing/api/v1/remote_shares/pending |
      | /ocs/v1.php/apps/files_sharing/api/v1/shares                |
      | /ocs/v1.php/config                                          |
      | /ocs/v1.php/privatedata/getattribute                        |
    Then the HTTP status code of responses on each endpoint should be "404,404,200,200,404" respectively
    When the user "Alice" requests these endpoints with "GET" with basic auth
      | endpoint                                                    |
      | /ocs/v2.php/apps/files_sharing/api/v1/remote_shares         |
      | /ocs/v2.php/apps/files_sharing/api/v1/remote_shares/pending |
      | /ocs/v2.php/apps/files_sharing/api/v1/shares                |
      | /ocs/v2.php/config                                          |
      | /ocs/v2.php/privatedata/getattribute                        |
    Then the HTTP status code of responses on each endpoint should be "404,404,200,200,404" respectively
    When the user "Alice" requests these endpoints with "GET" with basic auth
      | endpoint                 |
      | /ocs/v1.php/cloud/apps   |
      | /ocs/v1.php/cloud/groups |
      | /ocs/v1.php/cloud/users  |
      | /ocs/v2.php/cloud/apps   |
      | /ocs/v2.php/cloud/groups |
      | /ocs/v2.php/cloud/users  |
    Then the HTTP status code of responses on all endpoints should be "404"

  @owncloud-issue-1338 @owncloud-issue-1337 @smokeTest
  Scenario: using OCS as normal user with wrong password
    When user "Alice" requests these endpoints with "GET" using password "invalid"
      | endpoint                                                    |
      | /ocs/v1.php/apps/files_sharing/api/v1/remote_shares         |
      | /ocs/v2.php/apps/files_sharing/api/v1/remote_shares         |
      | /ocs/v1.php/apps/files_sharing/api/v1/remote_shares/pending |
      | /ocs/v2.php/apps/files_sharing/api/v1/remote_shares/pending |
      | /ocs/v1.php/apps/files_sharing/api/v1/shares                |
      | /ocs/v2.php/apps/files_sharing/api/v1/shares                |
      | /ocs/v1.php/cloud/apps                                      |
      | /ocs/v2.php/cloud/apps                                      |
      | /ocs/v1.php/cloud/groups                                    |
      | /ocs/v2.php/cloud/groups                                    |
      | /ocs/v1.php/cloud/users                                     |
      | /ocs/v2.php/cloud/users                                     |
      | /ocs/v1.php/privatedata/getattribute                        |
      | /ocs/v2.php/privatedata/getattribute                        |
    Then the HTTP status code of responses on all endpoints should be "401"
    When user "Alice" requests these endpoints with "GET" using password "invalid"
      | endpoint           |
      | /ocs/v1.php/config |
    Then the HTTP status code of responses on all endpoints should be "200"
    When user "Alice" requests these endpoints with "GET" using password "invalid"
      | endpoint           |
      | /ocs/v2.php/config |
    Then the HTTP status code of responses on all endpoints should be "200"

  @owncloud-issue-1319
  Scenario: using OCS with admin basic auth
    When the administrator requests these endpoints with "GET"
      | endpoint                 |
      | /ocs/v1.php/cloud/apps   |
      | /ocs/v1.php/cloud/groups |
      | /ocs/v1.php/cloud/users  |
    Then the HTTP status code of responses on all endpoints should be "404"
    When the administrator requests these endpoints with "GET"
      | endpoint                 |
      | /ocs/v2.php/cloud/apps   |
      | /ocs/v2.php/cloud/groups |
      | /ocs/v2.php/cloud/users  |
    Then the HTTP status code of responses on all endpoints should be "404"

  @owncloud-issue-1337 @owncloud-issue-1319
  Scenario: using OCS as admin user with wrong password
    When user "admin" requests these endpoints with "GET" using password "invalid"
      | endpoint                                                    |
      | /ocs/v1.php/apps/files_sharing/api/v1/remote_shares         |
      | /ocs/v2.php/apps/files_sharing/api/v1/remote_shares         |
      | /ocs/v1.php/apps/files_sharing/api/v1/remote_shares/pending |
      | /ocs/v2.php/apps/files_sharing/api/v1/remote_shares/pending |
      | /ocs/v1.php/apps/files_sharing/api/v1/shares                |
      | /ocs/v2.php/apps/files_sharing/api/v1/shares                |
      | /ocs/v1.php/cloud/apps                                      |
      | /ocs/v2.php/cloud/apps                                      |
      | /ocs/v1.php/cloud/groups                                    |
      | /ocs/v2.php/cloud/groups                                    |
      | /ocs/v1.php/cloud/users                                     |
      | /ocs/v2.php/cloud/users                                     |
      | /ocs/v1.php/privatedata/getattribute                        |
      | /ocs/v2.php/privatedata/getattribute                        |
    Then the HTTP status code of responses on all endpoints should be "401"
    When user "another-admin" requests these endpoints with "GET" using password "invalid"
      | endpoint           |
      | /ocs/v1.php/config |
    Then the HTTP status code of responses on all endpoints should be "200"
    When user "another-admin" requests these endpoints with "GET" using password "invalid"
      | endpoint           |
      | /ocs/v2.php/config |
    Then the HTTP status code of responses on all endpoints should be "200"
