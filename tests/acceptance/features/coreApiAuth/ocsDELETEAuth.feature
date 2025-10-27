Feature: auth
  As a user
  I want to send delete request with wrong password
  So that I can make sure it doesn't work

  @smokeTest @owncloud-issue-1337 @owncloud-issue-1319
  Scenario: send DELETE requests to OCS endpoints as admin with wrong password
    When user "admin" requests these endpoints with "DELETE" using password "invalid" about user "Alice"
      | endpoint                                                        |
      | /ocs/v1.php/apps/files_sharing/api/v1/remote_shares/pending/123 |
      | /ocs/v2.php/apps/files_sharing/api/v1/remote_shares/pending/123 |
      | /ocs/v1.php/apps/files_sharing/api/v1/remote_shares/123         |
      | /ocs/v2.php/apps/files_sharing/api/v1/remote_shares/123         |
      | /ocs/v2.php/apps/files_sharing/api/v1/shares/123                |
      | /ocs/v1.php/apps/files_sharing/api/v1/shares/pending/123        |
      | /ocs/v2.php/apps/files_sharing/api/v1/shares/pending/123        |
      | /ocs/v1.php/cloud/apps/testing                                  |
      | /ocs/v2.php/cloud/apps/testing                                  |
      | /ocs/v1.php/cloud/groups/group1                                 |
      | /ocs/v2.php/cloud/groups/group1                                 |
      | /ocs/v1.php/cloud/users/%username%                              |
      | /ocs/v2.php/cloud/users/%username%                              |
      | /ocs/v1.php/cloud/users/%username%/groups                       |
      | /ocs/v2.php/cloud/users/%username%/groups                       |
      | /ocs/v1.php/cloud/users/%username%/subadmins                    |
      | /ocs/v2.php/cloud/users/%username%/subadmins                    |
    Then the HTTP status code of responses on all endpoints should be "401"
