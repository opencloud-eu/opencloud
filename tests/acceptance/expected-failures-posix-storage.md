## Scenarios from OpenCloud API tests that are expected to fail with posix storage

#### [Downloading the archive of the resource (files | folder) using resource path is not possible](https://github.com/owncloud/ocis/issues/4637)

- [apiArchiver/downloadByPath.feature:25](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L25)
- [apiArchiver/downloadByPath.feature:26](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L26)
- [apiArchiver/downloadByPath.feature:43](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L43)
- [apiArchiver/downloadByPath.feature:44](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L44)
- [apiArchiver/downloadByPath.feature:47](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L47)
- [apiArchiver/downloadByPath.feature:73](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L73)
- [apiArchiver/downloadByPath.feature:163](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L163)
- [apiArchiver/downloadByPath.feature:164](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiArchiver/downloadByPath.feature#L164)

#### [PATCH request for TUS upload with wrong checksum gives incorrect response](https://github.com/owncloud/ocis/issues/1755)

- [apiSpacesShares/shareUploadTUS.feature:267](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/shareUploadTUS.feature#L267)
- [apiSpacesShares/shareUploadTUS.feature:286](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/shareUploadTUS.feature#L286)
- [apiSpacesShares/shareUploadTUS.feature:364](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/shareUploadTUS.feature#L364)

#### [Settings service user can list other peoples assignments](https://github.com/owncloud/ocis/issues/5032)

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

- [apiGraphUserGroup/addUserToGroup.feature:294](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/addUserToGroup.feature#L294)

#### [Users are added in a group with wrong host in host-part of user](https://github.com/owncloud/ocis/issues/5871)

- [apiGraphUserGroup/addUserToGroup.feature:378](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/addUserToGroup.feature#L378)
- [apiGraphUserGroup/addUserToGroup.feature:392](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiGraphUserGroup/addUserToGroup.feature#L392)

#### [Shared file locking is not possible using different path](https://github.com/owncloud/ocis/issues/7599)

- [apiLocks/lockFiles.feature:184](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L184)
- [apiLocks/lockFiles.feature:185](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L185)
- [apiLocks/lockFiles.feature:186](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L186)
- [apiLocks/lockFiles.feature:303](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L303)
- [apiLocks/lockFiles.feature:304](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L304)
- [apiLocks/lockFiles.feature:305](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L305)
- [apiLocks/lockFiles.feature:356](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L356)
- [apiLocks/lockFiles.feature:357](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L357)
- [apiLocks/lockFiles.feature:358](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L358)
- [apiLocks/lockFiles.feature:359](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L359)
- [apiLocks/lockFiles.feature:360](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L360)
- [apiLocks/lockFiles.feature:361](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L361)
- [apiLocks/lockFiles.feature:389](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L389)
- [apiLocks/lockFiles.feature:390](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L390)
- [apiLocks/lockFiles.feature:391](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L391)
- [apiLocks/lockFiles.feature:392](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L392)
- [apiLocks/lockFiles.feature:393](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L393)
- [apiLocks/lockFiles.feature:394](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L394)
- [apiLocks/unlockFiles.feature:62](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L62)
- [apiLocks/unlockFiles.feature:63](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L63)
- [apiLocks/unlockFiles.feature:64](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L64)
- [apiLocks/unlockFiles.feature:168](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L168)
- [apiLocks/unlockFiles.feature:169](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L169)
- [apiLocks/unlockFiles.feature:170](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L170)
- [apiLocks/unlockFiles.feature:171](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L171)
- [apiLocks/unlockFiles.feature:172](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L172)
- [apiLocks/unlockFiles.feature:173](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L173)
- [apiLocks/unlockFiles.feature:195](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L195)
- [apiLocks/unlockFiles.feature:196](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L196)
- [apiLocks/unlockFiles.feature:197](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L197)
- [apiLocks/unlockFiles.feature:198](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L198)
- [apiLocks/unlockFiles.feature:199](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L199)
- [apiLocks/unlockFiles.feature:200](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L200)
- [apiLocks/unlockFiles.feature:222](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L222)
- [apiLocks/unlockFiles.feature:223](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L223)
- [apiLocks/unlockFiles.feature:224](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L224)
- [apiLocks/unlockFiles.feature:225](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L225)
- [apiLocks/unlockFiles.feature:226](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L226)
- [apiLocks/unlockFiles.feature:227](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L227)

#### [Folders can be locked and locking works partially](https://github.com/owncloud/ocis/issues/7641)

- [apiLocks/lockFiles.feature:433](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L433)
- [apiLocks/lockFiles.feature:434](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L434)
- [apiLocks/lockFiles.feature:435](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L435)
- [apiLocks/lockFiles.feature:436](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L436)
- [apiLocks/lockFiles.feature:437](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L437)
- [apiLocks/lockFiles.feature:438](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L438)
- [apiLocks/lockFiles.feature:407](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L407)
- [apiLocks/lockFiles.feature:408](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L408)
- [apiLocks/lockFiles.feature:409](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L409)
- [apiLocks/lockFiles.feature:410](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L410)
- [apiLocks/lockFiles.feature:411](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L411)
- [apiLocks/lockFiles.feature:412](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L412)

#### [Anonymous users can unlock a file shared to them through a public link if they get the lock token](https://github.com/owncloud/ocis/issues/7761)

- [apiLocks/unlockFiles.feature:42](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L42)
- [apiLocks/unlockFiles.feature:43](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L43)
- [apiLocks/unlockFiles.feature:44](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L44)
- [apiLocks/unlockFiles.feature:45](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L45)
- [apiLocks/unlockFiles.feature:46](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L46)
- [apiLocks/unlockFiles.feature:47](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L47)

#### [Trying to unlock a shared file with sharer's lock token gives 500](https://github.com/owncloud/ocis/issues/7767)

- [apiLocks/unlockFiles.feature:114](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L114)
- [apiLocks/unlockFiles.feature:115](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L115)
- [apiLocks/unlockFiles.feature:116](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L116)
- [apiLocks/unlockFiles.feature:117](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L117)
- [apiLocks/unlockFiles.feature:118](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L118)
- [apiLocks/unlockFiles.feature:119](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L119)
- [apiLocks/unlockFiles.feature:141](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L141)
- [apiLocks/unlockFiles.feature:142](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L142)
- [apiLocks/unlockFiles.feature:143](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L143)
- [apiLocks/unlockFiles.feature:144](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L144)
- [apiLocks/unlockFiles.feature:145](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L145)
- [apiLocks/unlockFiles.feature:146](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/unlockFiles.feature#L146)

#### [Anonymous user trying lock a file shared to them through a public link gives 405](https://github.com/owncloud/ocis/issues/7790)

- [apiLocks/lockFiles.feature:522](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L522)
- [apiLocks/lockFiles.feature:523](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L523)
- [apiLocks/lockFiles.feature:524](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L524)
- [apiLocks/lockFiles.feature:525](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L525)
- [apiLocks/lockFiles.feature:544](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L544)
- [apiLocks/lockFiles.feature:545](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L545)
- [apiLocks/lockFiles.feature:546](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L546)
- [apiLocks/lockFiles.feature:547](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiLocks/lockFiles.feature#L547)

#### [sharee (editor role) MOVE a file by file-id into shared sub-folder returns 502](https://github.com/owncloud/ocis/issues/7617)

- [apiSpacesDavOperation/moveByFileId.feature:366](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L366)
- [apiSpacesDavOperation/moveByFileId.feature:609](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L609)

#### [MOVE a file into same folder with same name returns 404 instead of 403](https://github.com/owncloud/ocis/issues/1976)

- [apiSpacesShares/moveSpaces.feature:69](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/moveSpaces.feature#L69)
- [apiSpacesShares/moveSpaces.feature:70](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/moveSpaces.feature#L70)
- [apiSpacesShares/moveSpaces.feature:408](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesShares/moveSpaces.feature#L408)
- [apiSpacesDavOperation/moveByFileId.feature:61](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L61)
- [apiSpacesDavOperation/moveByFileId.feature:173](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L173)
- [apiSpacesDavOperation/moveByFileId.feature:174](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L174)
- [apiSpacesDavOperation/moveByFileId.feature:175](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L175)
- [apiSpacesDavOperation/moveByFileId.feature:390](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSpacesDavOperation/moveByFileId.feature#L390)

#### [OCM. admin cannot get federated users if he hasn't connection with them ](https://github.com/owncloud/ocis/issues/9829)

- [apiOcm/searchFederationUsers.feature:429](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiOcm/searchFederationUsers.feature#L429)
- [apiOcm/searchFederationUsers.feature:601](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiOcm/searchFederationUsers.feature#L601)

#### [OCM. federated connection is not dropped when one of the users deletes the connection](https://github.com/owncloud/ocis/issues/10216)

- [apiOcm/deleteFederatedConnections.feature:21](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiOcm/deleteFederatedConnections.feature#L21)
- [apiOcm/deleteFederatedConnections.feature:67](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiOcm/deleteFederatedConnections.feature#L67)

#### [OCM. server crash after deleting share for ocm user](https://github.com/owncloud/ocis/issues/10213)

- [apiOcm/deleteFederatedConnections.feature:102](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiOcm/deleteFederatedConnections.feature#L102)

#### [Readiness check for some services returns 500 status code](https://github.com/owncloud/ocis/issues/10661)

- [apiServiceAvailability/serviceAvailabilityCheck.feature:123](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiServiceAvailability/serviceAvailabilityCheck.feature#L123)

## Scenarios from core API tests that are expected to fail with posix storage

### File

Basic file management like up and download, move, copy, properties, trash, versions and chunking.

#### [Custom dav properties with namespaces are rendered incorrectly](https://github.com/owncloud/ocis/issues/2140)

_ocdav: double-check the webdav property parsing when custom namespaces are used_

- [coreApiWebdavProperties/setFileProperties.feature:127](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/setFileProperties.feature#L127)
- [coreApiWebdavProperties/setFileProperties.feature:128](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/setFileProperties.feature#L128)
- [coreApiWebdavProperties/setFileProperties.feature:129](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/setFileProperties.feature#L129)

### Sync

Synchronization features like etag propagation, setting mtime and locking files

#### [Uploading an old method chunked file with checksum should fail using new DAV path](https://github.com/owncloud/ocis/issues/2323)

- [coreApiMain/checksums.feature:231](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiMain/checksums.feature#L231)
- [coreApiMain/checksums.feature:232](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiMain/checksums.feature#L232)
- [coreApiMain/checksums.feature:233](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiMain/checksums.feature#L233)

### Share

#### [d:quota-available-bytes in dprop of PROPFIND give wrong response value](https://github.com/owncloud/ocis/issues/8197)

- [coreApiWebdavProperties/getQuota.feature:56](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L56)
- [coreApiWebdavProperties/getQuota.feature:57](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L57)
- [coreApiWebdavProperties/getQuota.feature:58](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L58)
- [coreApiWebdavProperties/getQuota.feature:72](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L72)
- [coreApiWebdavProperties/getQuota.feature:73](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L73)
- [coreApiWebdavProperties/getQuota.feature:74](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavProperties/getQuota.feature#L74)

#### [deleting a file inside a received shared folder is moved to the trash-bin of the sharer not the receiver](https://github.com/owncloud/ocis/issues/1124)

- [coreApiTrashbin/trashbinSharingToShares.feature:52](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L52)
- [coreApiTrashbin/trashbinSharingToShares.feature:53](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L53)
- [coreApiTrashbin/trashbinSharingToShares.feature:54](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L54)
- [coreApiTrashbin/trashbinSharingToShares.feature:79](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L79)
- [coreApiTrashbin/trashbinSharingToShares.feature:80](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L80)
- [coreApiTrashbin/trashbinSharingToShares.feature:81](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L81)
- [coreApiTrashbin/trashbinSharingToShares.feature:134](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L134)
- [coreApiTrashbin/trashbinSharingToShares.feature:135](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L135)
- [coreApiTrashbin/trashbinSharingToShares.feature:136](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L136)
- [coreApiTrashbin/trashbinSharingToShares.feature:191](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L191)
- [coreApiTrashbin/trashbinSharingToShares.feature:192](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L192)

### Other

API, search, favorites, config, capabilities, not existing endpoints, CORS and others

#### [sending MKCOL requests to another or non-existing user's webDav endpoints as normal user should return 404](https://github.com/owncloud/ocis/issues/5049)

_ocdav: api compatibility, return correct status code_

- [coreApiAuth/webDavMKCOLAuth.feature:42](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiAuth/webDavMKCOLAuth.feature#L42)
- [coreApiAuth/webDavMKCOLAuth.feature:53](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiAuth/webDavMKCOLAuth.feature#L53)

#### [trying to lock file of another user gives http 500](https://github.com/owncloud/ocis/issues/2176)

- [coreApiAuth/webDavLOCKAuth.feature:46](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiAuth/webDavLOCKAuth.feature#L46)
- [coreApiAuth/webDavLOCKAuth.feature:58](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiAuth/webDavLOCKAuth.feature#L58)



#### [WWW-Authenticate header for unauthenticated requests is not clear](https://github.com/owncloud/ocis/issues/2285)

- [coreApiWebdavOperations/refuseAccess.feature:21](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavOperations/refuseAccess.feature#L21)
- [coreApiWebdavOperations/refuseAccess.feature:22](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavOperations/refuseAccess.feature#L22)

#### [PATCH request for TUS upload with wrong checksum gives incorrect response](https://github.com/owncloud/ocis/issues/1755)

- [coreApiWebdavUploadTUS/checksums.feature:74](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L74)
- [coreApiWebdavUploadTUS/checksums.feature:75](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L75)
- [coreApiWebdavUploadTUS/checksums.feature:76](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L76)
- [coreApiWebdavUploadTUS/checksums.feature:77](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L77)
- [coreApiWebdavUploadTUS/checksums.feature:79](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L79)
- [coreApiWebdavUploadTUS/checksums.feature:78](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L78)
- [coreApiWebdavUploadTUS/checksums.feature:147](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L147)
- [coreApiWebdavUploadTUS/checksums.feature:148](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L148)
- [coreApiWebdavUploadTUS/checksums.feature:149](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L149)
- [coreApiWebdavUploadTUS/checksums.feature:192](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L192)
- [coreApiWebdavUploadTUS/checksums.feature:193](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L193)
- [coreApiWebdavUploadTUS/checksums.feature:194](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L194)
- [coreApiWebdavUploadTUS/checksums.feature:195](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L195)
- [coreApiWebdavUploadTUS/checksums.feature:196](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L196)
- [coreApiWebdavUploadTUS/checksums.feature:197](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L197)
- [coreApiWebdavUploadTUS/checksums.feature:240](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L240)
- [coreApiWebdavUploadTUS/checksums.feature:241](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L241)
- [coreApiWebdavUploadTUS/checksums.feature:242](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L242)
- [coreApiWebdavUploadTUS/checksums.feature:243](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L243)
- [coreApiWebdavUploadTUS/checksums.feature:244](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L244)
- [coreApiWebdavUploadTUS/checksums.feature:245](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/checksums.feature#L245)
- [coreApiWebdavUploadTUS/uploadToShare.feature:244](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L244)
- [coreApiWebdavUploadTUS/uploadToShare.feature:245](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L245)
- [coreApiWebdavUploadTUS/uploadToShare.feature:267](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L267)
- [coreApiWebdavUploadTUS/uploadToShare.feature:268](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L268)
- [coreApiWebdavUploadTUS/uploadToShare.feature:361](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L361)
- [coreApiWebdavUploadTUS/uploadToShare.feature:362](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavUploadTUS/uploadToShare.feature#L362)


#### [Trying to delete other user's trashbin item returns 409 for spaces path instead of 404](https://github.com/owncloud/ocis/issues/9791)

- [coreApiTrashbin/trashbinDelete.feature:92](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinDelete.feature#L92)

#### [MOVE a file into same folder with same name returns 404 instead of 403](https://github.com/owncloud/ocis/issues/1976)

- [coreApiWebdavMove/moveFile.feature:100](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove/moveFile.feature#L100)
- [coreApiWebdavMove/moveFile.feature:101](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove/moveFile.feature#L101)
- [coreApiWebdavMove/moveFile.feature:102](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove/moveFile.feature#L102)
- [coreApiWebdavMove/moveFolder.feature:217](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove/moveFolder.feature#L217)
- [coreApiWebdavMove/moveFolder.feature:218](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove/moveFolder.feature#L218)
- [coreApiWebdavMove/moveFolder.feature:219](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove/moveFolder.feature#L219)
- [coreApiWebdavMove/moveShareOnOpencloud.feature:323](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove/moveShareOnOpencloud.feature#L323)
- [coreApiWebdavMove/moveShareOnOpencloud.feature:326](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove/moveShareOnOpencloud.feature#L326)
- [coreApiWebdavMove/moveShareOnOpencloud.feature:329](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavMove/moveShareOnOpencloud.feature#L329)

#### [COPY file/folder to same name is possible (but 500 code error for folder with spaces path)](https://github.com/owncloud/ocis/issues/8711)

- [coreApiSharePublicLink2/copyFromPublicLink.feature:198](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiSharePublicLink2/copyFromPublicLink.feature#L198)
- [coreApiWebdavCopyCreate/copyFile.feature:1064](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavCopyCreate/copyFile.feature#L1064)
- [coreApiWebdavCopyCreate/copyFile.feature:1065](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavCopyCreate/copyFile.feature#L1065)
- [coreApiWebdavCopyCreate/copyFile.feature:1066](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiWebdavCopyCreate/copyFile.feature#L1066)

#### [Trying to restore personal file to file of share received folder returns 403 but the share file is deleted (new dav path)](https://github.com/owncloud/ocis/issues/10356)

- [coreApiTrashbin/trashbinSharingToShares.feature:263](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/coreApiTrashbin/trashbinSharingToShares.feature#L263)

#### [Report response doesn't contain <d:getetag>](https://github.com/opencloud-eu/opencloud/issues/3501)

- [apiSearch1/search.feature:452](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L452)
- [apiSearch1/search.feature:453](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L453)
- [apiSearch1/search.feature:454](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L454)
- [apiSearch1/search.feature:476](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L476)
- [apiSearch1/search.feature:477](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L477)
- [apiSearch1/search.feature:478](https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/apiSearch1/search.feature#L478)

### Won't fix

Not everything needs to be implemented for opencloud.

- _Blacklisted ignored files are no longer required because opencloud can handle `.htaccess` files without security implications introduced by serving user provided files with apache._

Note: always have an empty line at the end of this file.
The bash script that processes this file requires that the last line has a newline on the end.
