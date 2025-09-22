## Scenarios from OpenCloud API tests that are expected to fail with decomposed storage

#### [Downloading the archive of the resource (files | folder) using resource path is not possible](https://github.com/owncloud/ocis/issues/4637)

- [apiArchiver/downloadByPath.feature:25](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L25)
- [apiArchiver/downloadByPath.feature:26](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L26)
- [apiArchiver/downloadByPath.feature:43](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L43)
- [apiArchiver/downloadByPath.feature:44](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L44)
- [apiArchiver/downloadByPath.feature:47](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L47)
- [apiArchiver/downloadByPath.feature:73](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L73)
- [apiArchiver/downloadByPath.feature:171](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L171)
- [apiArchiver/downloadByPath.feature:172](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L172)

#### [PATCH request for TUS upload with wrong checksum gives incorrect response](https://github.com/owncloud/ocis/issues/1755)

- [apiSpacesShares/shareUploadTUS.feature:283](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/shareUploadTUS.feature#L283)
- [apiSpacesShares/shareUploadTUS.feature:303](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/shareUploadTUS.feature#L303)
- [apiSpacesShares/shareUploadTUS.feature:384](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/shareUploadTUS.feature#L384)

#### [Settings service user can list other peoples assignments](https://github.com/owncloud/ocis/issues/5032)

- [apiAccountsHashDifficulty/assignRole.feature:27](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiAccountsHashDifficulty/assignRole.feature#L27)
- [apiAccountsHashDifficulty/assignRole.feature:28](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiAccountsHashDifficulty/assignRole.feature#L28)
- [apiGraph/getAssignedRole.feature:31](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraph/getAssignedRole.feature#L31)
- [apiGraph/getAssignedRole.feature:32](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraph/getAssignedRole.feature#L32)
- [apiGraph/getAssignedRole.feature:33](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraph/getAssignedRole.feature#L33)

#### [A User can get information of another user with Graph API](https://github.com/owncloud/ocis/issues/5125)

- [apiGraphUserGroup/getUser.feature:84](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getUser.feature#L84)
- [apiGraphUserGroup/getUser.feature:85](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getUser.feature#L85)
- [apiGraphUserGroup/getUser.feature:86](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getUser.feature#L86)
- [apiGraphUserGroup/getUser.feature:628](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getUser.feature#L628)
- [apiGraphUserGroup/getUser.feature:629](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getUser.feature#L629)
- [apiGraphUserGroup/getUser.feature:630](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getUser.feature#L630)
- [apiGraphUserGroup/getUser.feature:645](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getUser.feature#L645)
- [apiGraphUserGroup/getUser.feature:646](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getUser.feature#L646)
- [apiGraphUserGroup/getUser.feature:647](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getUser.feature#L647)

#### [Normal user can get expanded members information of a group](https://github.com/owncloud/ocis/issues/5604)

- [apiGraphUserGroup/getGroup.feature:399](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getGroup.feature#L399)
- [apiGraphUserGroup/getGroup.feature:400](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getGroup.feature#L400)
- [apiGraphUserGroup/getGroup.feature:401](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getGroup.feature#L401)
- [apiGraphUserGroup/getGroup.feature:460](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getGroup.feature#L460)
- [apiGraphUserGroup/getGroup.feature:461](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getGroup.feature#L461)
- [apiGraphUserGroup/getGroup.feature:462](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getGroup.feature#L462)
- [apiGraphUserGroup/getGroup.feature:508](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getGroup.feature#L508)
- [apiGraphUserGroup/getGroup.feature:509](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getGroup.feature#L509)
- [apiGraphUserGroup/getGroup.feature:510](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/getGroup.feature#L510)

#### [Same users can be added in a group multiple time](https://github.com/owncloud/ocis/issues/5702)

- [apiGraphUserGroup/addUserToGroup.feature:295](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/addUserToGroup.feature#L295)

#### [Users are added in a group with wrong host in host-part of user](https://github.com/owncloud/ocis/issues/5871)

- [apiGraphUserGroup/addUserToGroup.feature:379](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/addUserToGroup.feature#L379)
- [apiGraphUserGroup/addUserToGroup.feature:393](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/addUserToGroup.feature#L393)

#### [Adding the same user as multiple members in a single request results in listing the same user twice in the group](https://github.com/owncloud/ocis/issues/5855)

- [apiGraphUserGroup/addUserToGroup.feature:430](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/addUserToGroup.feature#L430)

#### [Shared file locking is not possible using different path](https://github.com/owncloud/ocis/issues/7599)

- [apiLocks/lockFiles.feature:185](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L185)
- [apiLocks/lockFiles.feature:186](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L186)
- [apiLocks/lockFiles.feature:187](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L187)
- [apiLocks/lockFiles.feature:309](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L309)
- [apiLocks/lockFiles.feature:310](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L310)
- [apiLocks/lockFiles.feature:311](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L311)
- [apiLocks/lockFiles.feature:364](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L364)
- [apiLocks/lockFiles.feature:365](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L365)
- [apiLocks/lockFiles.feature:366](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L366)
- [apiLocks/lockFiles.feature:367](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L367)
- [apiLocks/lockFiles.feature:368](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L368)
- [apiLocks/lockFiles.feature:369](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L369)
- [apiLocks/lockFiles.feature:399](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L399)
- [apiLocks/lockFiles.feature:400](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L400)
- [apiLocks/lockFiles.feature:401](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L401)
- [apiLocks/lockFiles.feature:402](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L402)
- [apiLocks/lockFiles.feature:403](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L403)
- [apiLocks/lockFiles.feature:404](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L404)
- [apiLocks/unlockFiles.feature:62](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L62)
- [apiLocks/unlockFiles.feature:63](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L63)
- [apiLocks/unlockFiles.feature:64](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L64)
- [apiLocks/unlockFiles.feature:171](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L171)
- [apiLocks/unlockFiles.feature:172](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L172)
- [apiLocks/unlockFiles.feature:173](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L173)
- [apiLocks/unlockFiles.feature:174](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L174)
- [apiLocks/unlockFiles.feature:175](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L175)
- [apiLocks/unlockFiles.feature:176](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L176)
- [apiLocks/unlockFiles.feature:199](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L199)
- [apiLocks/unlockFiles.feature:200](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L200)
- [apiLocks/unlockFiles.feature:201](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L201)
- [apiLocks/unlockFiles.feature:202](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L202)
- [apiLocks/unlockFiles.feature:203](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L203)
- [apiLocks/unlockFiles.feature:204](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L204)
- [apiLocks/unlockFiles.feature:227](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L227)
- [apiLocks/unlockFiles.feature:228](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L228)
- [apiLocks/unlockFiles.feature:229](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L229)
- [apiLocks/unlockFiles.feature:230](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L230)
- [apiLocks/unlockFiles.feature:231](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L231)
- [apiLocks/unlockFiles.feature:232](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L232)

#### [Folders can be locked and locking works partially](https://github.com/owncloud/ocis/issues/7641)

- [apiLocks/lockFiles.feature:443](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L443)
- [apiLocks/lockFiles.feature:444](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L444)
- [apiLocks/lockFiles.feature:445](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L445)
- [apiLocks/lockFiles.feature:446](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L446)
- [apiLocks/lockFiles.feature:447](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L447)
- [apiLocks/lockFiles.feature:448](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L448)
- [apiLocks/lockFiles.feature:417](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L417)
- [apiLocks/lockFiles.feature:418](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L418)
- [apiLocks/lockFiles.feature:419](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L419)
- [apiLocks/lockFiles.feature:420](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L420)
- [apiLocks/lockFiles.feature:421](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L421)
- [apiLocks/lockFiles.feature:422](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L422)

#### [Anonymous users can unlock a file shared to them through a public link if they get the lock token](https://github.com/owncloud/ocis/issues/7761)

- [apiLocks/unlockFiles.feature:42](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L42)
- [apiLocks/unlockFiles.feature:43](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L43)
- [apiLocks/unlockFiles.feature:44](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L44)
- [apiLocks/unlockFiles.feature:45](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L45)
- [apiLocks/unlockFiles.feature:46](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L46)
- [apiLocks/unlockFiles.feature:47](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L47)

#### [Trying to unlock a shared file with sharer's lock token gives 500](https://github.com/owncloud/ocis/issues/7767)

- [apiLocks/unlockFiles.feature:115](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L115)
- [apiLocks/unlockFiles.feature:116](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L116)
- [apiLocks/unlockFiles.feature:117](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L117)
- [apiLocks/unlockFiles.feature:118](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L118)
- [apiLocks/unlockFiles.feature:119](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L119)
- [apiLocks/unlockFiles.feature:120](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L120)
- [apiLocks/unlockFiles.feature:143](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L143)
- [apiLocks/unlockFiles.feature:144](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L144)
- [apiLocks/unlockFiles.feature:145](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L145)
- [apiLocks/unlockFiles.feature:146](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L146)
- [apiLocks/unlockFiles.feature:147](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L147)
- [apiLocks/unlockFiles.feature:148](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L148)

#### [Anonymous user trying lock a file shared to them through a public link gives 405](https://github.com/owncloud/ocis/issues/7790)

- [apiLocks/lockFiles.feature:532](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L532)
- [apiLocks/lockFiles.feature:533](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L533)
- [apiLocks/lockFiles.feature:534](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L534)
- [apiLocks/lockFiles.feature:535](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L535)
- [apiLocks/lockFiles.feature:554](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L554)
- [apiLocks/lockFiles.feature:555](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L555)
- [apiLocks/lockFiles.feature:556](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L556)
- [apiLocks/lockFiles.feature:557](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L557)

#### [sharee (editor role) MOVE a file by file-id into shared sub-folder returns 502](https://github.com/owncloud/ocis/issues/7617)

- [apiSpacesDavOperation/moveByFileId.feature:368](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L368)
- [apiSpacesDavOperation/moveByFileId.feature:591](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L591)

#### [MOVE a file into same folder with same name returns 404 instead of 403](https://github.com/owncloud/ocis/issues/1976)

- [apiSpacesShares/moveSpaces.feature:69](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/moveSpaces.feature#L69)
- [apiSpacesShares/moveSpaces.feature:70](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/moveSpaces.feature#L70)
- [apiSpacesShares/moveSpaces.feature:416](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/moveSpaces.feature#L416)
- [apiSpacesDavOperation/moveByFileId.feature:61](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L61)
- [apiSpacesDavOperation/moveByFileId.feature:174](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L174)
- [apiSpacesDavOperation/moveByFileId.feature:175](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L175)
- [apiSpacesDavOperation/moveByFileId.feature:176](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L176)
- [apiSpacesDavOperation/moveByFileId.feature:393](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L393)

#### [OCM. admin cannot get federated users if he hasn't connection with them ](https://github.com/owncloud/ocis/issues/9829)

- [apiOcm/searchFederationUsers.feature:429](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiOcm/searchFederationUsers.feature#L429)
- [apiOcm/searchFederationUsers.feature:601](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiOcm/searchFederationUsers.feature#L601)

#### [OCM. federated connection is not dropped when one of the users deletes the connection](https://github.com/owncloud/ocis/issues/10216)

- [apiOcm/deleteFederatedConnections.feature:21](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiOcm/deleteFederatedConnections.feature#L21)
- [apiOcm/deleteFederatedConnections.feature:67](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiOcm/deleteFederatedConnections.feature#L67)

#### [OCM. server crash after deleting share for ocm user](https://github.com/owncloud/ocis/issues/10213)

- [apiOcm/deleteFederatedConnections.feature:102](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiOcm/deleteFederatedConnections.feature#L102)

#### [Shares Jail PROPFIND returns different File IDs for the same item](https://github.com/owncloud/ocis/issues/9933)

- [apiSharingNg1/propfindShares.feature:149](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSharingNg1/propfindShares.feature#L149)

#### [Readiness check for some services returns 500 status code](https://github.com/owncloud/ocis/issues/10661)

- [apiServiceAvailability/serviceAvailabilityCheck.feature:125](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiServiceAvailability/serviceAvailabilityCheck.feature#L125)

#### [Skip tests for different languages](https://github.com/opencloud-eu/opencloud/issues/183)
- [apiActivities/activities.feature:2598](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiActivities/activities.feature#L2598)


#### [Missing properties in REPORT response](https://github.com/owncloud/ocis/issues/9780), [d:getetag property has empty value in REPORT response](https://github.com/owncloud/ocis/issues/9783)

- [apiSearch1/search.feature:437](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L437)
- [apiSearch1/search.feature:438](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L438)
- [apiSearch1/search.feature:439](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L439)
- [apiSearch1/search.feature:465](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L465)
- [apiSearch1/search.feature:466](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L466)
- [apiSearch1/search.feature:467](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L467)

Note: always have an empty line at the end of this file.
The bash script that processes this file requires that the last line has a newline on the end.
