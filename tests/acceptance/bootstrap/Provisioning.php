<?php declare(strict_types=1);
/**
 * @author Sergio Bertolin <sbertolin@owncloud.com>
 *
 * @copyright Copyright (c) 2018, ownCloud GmbH
 * @license AGPL-3.0
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

use Behat\Gherkin\Node\TableNode;
use GuzzleHttp\Exception\ClientException;
use GuzzleHttp\Exception\GuzzleException;
use Psr\Http\Message\ResponseInterface;
use PHPUnit\Framework\Assert;
use TestHelpers\UserHelper;
use TestHelpers\HttpRequestHelper;
use TestHelpers\OcHelper;
use TestHelpers\WebDavHelper;
use TestHelpers\GraphHelper;
use Laminas\Ldap\Exception\LdapException;
use Laminas\Ldap\Ldap;
use TestHelpers\TokenHelper;

/**
 * Functions for provisioning of users and groups
 */
trait Provisioning {
	/**
	 * list of users that were created on the local server during test runs
	 * key is the lowercase username, value is an array of user attributes
	 */
	private array $createdUsers = [];

	/**
	 * list of users that were created on the remote server during test runs
	 * key is the lowercase username, value is an array of user attributes
	 */
	private array $createdRemoteUsers = [];
	private array $startingGroups = [];
	private array $createdRemoteGroups = [];
	private array $createdGroups = [];

	/**
	 * Check if this is the admin group.
	 *
	 * @param string $groupname
	 *
	 * @return boolean
	 */
	public function isLocalAdminGroup(string $groupname): bool {
		return ($groupname === "admin");
	}

	/**
	 * Usernames are not case-sensitive, and can generally be specified with any
	 * mix of upper and lower case. For remembering usernames use the normalized
	 * form so that "alice" and "Alice" are remembered as the same user.
	 *
	 * @param string|null $username
	 *
	 * @return string
	 */
	public function normalizeUsername(?string $username): string {
		return \strtolower((string)$username);
	}

	/**
	 * @return array
	 */
	public function getCreatedUsers(): array {
		return $this->createdUsers;
	}

	/**
	 * @return array
	 */
	public function getAllCreatedUsers(): array {
		return array_merge($this->createdUsers, $this->createdRemoteUsers);
	}

	/**
	 * @return array
	 */
	public function getCreatedGroups(): array {
		return $this->createdGroups;
	}

	/**
	 * returns the display name of a user
	 * if no "Display Name" is set the username is returned instead
	 *
	 * @param string $username
	 *
	 * @return string
	 */
	public function getUserDisplayName(string $username): string {
		$normalizedUsername = $this->normalizeUsername($username);
		$users = $this->getAllCreatedUsers();
		if (isset($users[$normalizedUsername]['displayname'])) {
			$displayName = (string) $users[$normalizedUsername]['displayname'];
			if ($displayName !== '') {
				return $displayName;
			}
		}
		return $username;
	}

	/**
	 * @param string $user
	 * @param string $attribute
	 *
	 * @return mixed
	 * @throws Exception
	 */
	public function getAttributeOfCreatedUser(string $user, string $attribute) {
		$usersList = $this->getAllCreatedUsers();
		$normalizedUsername = $this->normalizeUsername($user);
		if (\array_key_exists($normalizedUsername, $usersList)) {
			if (\array_key_exists($attribute, $usersList[$normalizedUsername])) {
				return $usersList[$normalizedUsername][$attribute];
			} else {
				throw new Exception(
					__METHOD__ . ": User '$user' has no attribute with name '$attribute'."
				);
			}
		} else {
			return false;
		}
	}

	/**
	 * @param string $group
	 * @param string $attribute
	 *
	 * @return mixed
	 * @throws Exception
	 */
	public function getAttributeOfCreatedGroup(string $group, string $attribute) {
		$groupsList = $this->getCreatedGroups();
		if (\array_key_exists($group, $groupsList)) {
			if (\array_key_exists($attribute, $groupsList[$group])) {
				return $groupsList[$group][$attribute];
			} else {
				throw new Exception(
					__METHOD__ . ": Group '$group' has no attribute with name '$attribute'."
				);
			}
		} else {
			return false;
		}
	}

	/**
	 *
	 * @param string $username
	 *
	 * @return string password
	 * @throws Exception
	 */
	public function getUserPassword(string $username): string {
		$normalizedUsername = $this->normalizeUsername($username);
		if ($normalizedUsername === $this->getAdminUsername()) {
			$password = $this->getAdminPassword();
		} elseif (\array_key_exists($normalizedUsername, $this->createdUsers)) {
			$password = $this->createdUsers[$normalizedUsername]['password'];
		} elseif (\array_key_exists($normalizedUsername, $this->createdRemoteUsers)) {
			$password = $this->createdRemoteUsers[$normalizedUsername]['password'];
		} else {
			throw new Exception(
				"user '$username' was not created by this test run"
			);
		}

		//make sure the function always returns a string
		return (string) $password;
	}

	/**
	 * @Given user :user has been created with default attributes
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userHasBeenCreatedWithDefaultAttributes(
		string $user
	): void {
		$this->userHasBeenCreated(["userName" => $user]);
	}

	/**
	 * @Given these users have been created without being initialized:
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userHasBeenCreatedWithDefaultAttributesAndNotInitialized(
		TableNode $table
	): void {
		$this->usersHaveBeenCreated($table, true, false);
	}

	/**
	 * @Given these users have been created with default attributes:
	 * expects a table of users with the heading
	 * "|username|"
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function theseUsersHaveBeenCreatedWithDefaultAttributesAndWithoutSkeletonFiles(TableNode $table): void {
		$this->usersHaveBeenCreated($table);
	}

	/**
	 * @Given the user :byUser has created a new user with the following attributes:
	 *
	 * @param string $byUser
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function theAdministratorHasCreatedANewUserWithFollowingSettings(string $byUser, TableNode $table): void {
		$rows = $table->getRowsHash();
		$this->userHasBeenCreated(
			$rows,
			$byUser
		);
	}

	/**
	 *
	 * @param string $groupname
	 *
	 * @return boolean
	 * @throws Exception
	 */
	public function theGroupShouldBeAbleToBeDeleted(string $groupname): bool {
		if (\array_key_exists($groupname, $this->createdGroups)) {
			return $this->createdGroups[$groupname]['possibleToDelete'] ?? true;
		}

		if (\array_key_exists($groupname, $this->createdRemoteGroups)) {
			return $this->createdRemoteGroups[$groupname]['possibleToDelete'] ?? true;
		}

		throw new Exception(
			__METHOD__
			. " group '$groupname' was not created by this test run"
		);
	}

	/**
	 *
	 * @param string $path
	 *
	 * @return void
	 */
	public function importLdifFile(string $path): void {
		$ldifData = \file_get_contents($path);
		$this->importLdifData($ldifData);
	}

	/**
	 * imports an ldif string
	 *
	 * @param string $ldifData
	 *
	 * @return void
	 */
	public function importLdifData(string $ldifData): void {
		$items = Laminas\Ldap\Ldif\Encoder::decode($ldifData);
		if (isset($items['dn'])) {
			//only one item in the ldif data
			$this->ldap->add($items['dn'], $items);
		} else {
			foreach ($items as $item) {
				if (isset($item["objectclass"])) {
					if (\in_array("posixGroup", $item["objectclass"])) {
						$this->ldapCreatedGroups[] = $item["cn"][0];
						$this->addGroupToCreatedGroupsList($item["cn"][0]);
					} elseif (\in_array("inetOrgPerson", $item["objectclass"])) {
						$this->ldapCreatedUsers[] = $item["uid"][0];
						$this->addUserToCreatedUsersList($item["uid"][0], $item["userpassword"][0]);
					}
				}
				$this->ldap->add($item['dn'], $item);
			}
		}
	}

	/**
	 * @param array $suiteParameters
	 *
	 * @return void
	 * @throws Exception
	 * @throws \LdapException
	 */
	public function connectToLdap(array $suiteParameters): void {
		$this->ldapBaseDN = OcHelper::getBaseDN();
		$this->ldapUsersOU = OcHelper::getUsersOU();
		$this->ldapGroupsOU = OcHelper::getGroupsOU();
		$this->ldapGroupSchema = OcHelper::getGroupSchema();
		$this->ldapHost = OcHelper::getHostname();
		$this->ldapPort = OcHelper::getLdapPort();
		$useSsl = OcHelper::useSsl();
		$this->ldapAdminUser = OcHelper::getBindDN();
		$this->ldapAdminPassword = OcHelper::getBindPassword();
		$this->skipImportLdif = (\getenv("REVA_LDAP_SKIP_LDIF_IMPORT") === "true");
		if ($useSsl === true) {
			\putenv('LDAPTLS_REQCERT=never');
		}

		if ($this->ldapAdminPassword === "") {
			$this->ldapAdminPassword = (string)$suiteParameters['ldapAdminPassword'];
		}
		$options = [
			'host' => $this->ldapHost,
			'port' => $this->ldapPort,
			'password' => $this->ldapAdminPassword,
			'bindRequiresDn' => true,
			'useSsl' => $useSsl,
			'baseDn' => $this->ldapBaseDN,
			'username' => $this->ldapAdminUser
		];
		$this->ldap = new Ldap($options);
		$this->ldap->bind();

		$ldifFile = __DIR__ . $suiteParameters['ldapInitialUserFilePath'];
		if (!$this->skipImportLdif) {
			try {
				$this->importLdifFile($ldifFile);
			} catch (LdapException $err) {
				if (!\str_contains($err->getMessage(), "Already exists")) {
					throw $err;
				}
			}
		}
	}

	/**
	 * prepares a suitable nested array with user-attributes for multiple users to be created
	 *
	 * @param boolean $setDefaultAttributes
	 * @param array $table
	 *
	 * @return array
	 * @throws JsonException
	 */
	public function buildUsersAttributesArray(bool $setDefaultAttributes, array $table): array {
		$usersAttributes = [];
		foreach ($table as $row) {
			$userAttribute['userid'] = $this->getActualUsername($row['username']);

			if (isset($row['displayname'])) {
				$userAttribute['displayName'] = $row['displayname'];
			} elseif ($setDefaultAttributes) {
				$userAttribute['displayName'] = $this->getDisplayNameForUser($row['username']);
				if ($userAttribute['displayName'] === null) {
					$userAttribute['displayName'] = $this->getDisplayNameForUser('regularuser');
				}
			} else {
				$userAttribute['displayName'] = null;
			}
			if (isset($row['email'])) {
				$userAttribute['email'] = $row['email'];
			} elseif ($setDefaultAttributes) {
				$userAttribute['email'] = $this->getEmailAddressForUser($row['username']);
				if ($userAttribute['email'] === null) {
					$userAttribute['email'] = $row['username'] . '@opencloud.eu';
				}
			} else {
				$userAttribute['email'] = null;
			}

			if (isset($row['password'])) {
				$userAttribute['password'] = $this->getActualPassword($row['password']);
			} else {
				$userAttribute['password'] = $this->getPasswordForUser($row['username']);
			}
			// Add request body to the bodies array. We will use that later to loop through created users.
			$usersAttributes[] = $userAttribute;
		}
		return $usersAttributes;
	}

	/**
	 * creates a user in the ldap server
	 * the created user is added to `createdUsersList`
	 * ldap users are re-synced after creating a new user
	 *
	 * @param array $setting
	 *
	 * @return void
	 * @throws Exception
	 */
	public function createLdapUser(array $setting): void {
		$ou =  $this->ldapUsersOU ;
		// Some special characters need to be escaped in LDAP DN and attributes
		// The special characters allowed in a username (UID) are +_.@-
		// Of these, only + has to be escaped.
		$userId = \str_replace('+', '\+', $setting["userid"]);
		$newDN = 'uid=' . $userId . ',ou=' . $ou . ',' . $this->ldapBaseDN;

		//pick a high uid number to make sure there are no conflicts with existing uid numbers
		$uidNumber = \count($this->ldapCreatedUsers) + 30000;
		$entry = [];
		$entry['cn'] = $userId;
		$entry['sn'] = $userId;
		$entry['uid'] = $setting["userid"];
		$entry['homeDirectory'] = '/home/openldap/' . $setting["userid"];
		$entry['objectclass'][] = 'posixAccount';
		$entry['objectclass'][] = 'inetOrgPerson';
		$entry['objectclass'][] = 'organizationalPerson';
		$entry['objectclass'][] = 'person';
		$entry['objectclass'][] = 'top';

		$entry['userPassword'] = $setting["password"];
		if (isset($setting["displayName"])) {
			$entry['displayName'] = $setting["displayName"];
		}
		if (isset($setting["email"])) {
			$entry['mail'] = $setting["email"];
		} elseif (!OcHelper::isTestingOnReva()) {
			$entry['mail'] = $userId . '@opencloud.eu';
		}
		$entry['gidNumber'] = 5000;
		$entry['uidNumber'] = $uidNumber;

		if (!OcHelper::isTestingOnReva()) {
			$entry['objectclass'][] = 'openCloudObject';
			$entry['openCloudUUID'] = WebDavHelper::generateUUIDv4();
		}

		try {
			$this->ldap->add($newDN, $entry);
		} catch (LdapException $e) {
			if (\str_contains($e->getMessage(), "Already exists")) {
				$this->ldap->delete(
					"uid=" . ldap_escape(
						$entry['uid'],
						"",
						LDAP_ESCAPE_DN
					) . ",ou=" . $this->ldapUsersOU . "," . $this->ldapBaseDN,
				);
				OcHelper::deleteRevaUserData([$entry['uid']]);
				$this->ldap->add($newDN, $entry);
			}
		}

		$this->ldapCreatedUsers[] = $setting["userid"];
	}

	/**
	 * @param string $group group name
	 *
	 * @return void
	 * @throws Exception
	 * @throws LdapException
	 */
	public function createLdapGroup(string $group): void {
		$baseDN = $this->getLdapBaseDN();
		$newDN = 'cn=' . $group . ',ou=' . $this->ldapGroupsOU . ',' . $baseDN;
		$entry = [];
		$entry['cn'] = $group;
		$entry['objectclass'][] = 'top';

		if ($this->ldapGroupSchema == "rfc2307") {
			$entry['objectclass'][] = 'posixGroup';
			$entry['gidNumber'] = 5000;
		} else {
			$entry['objectclass'][] = 'groupOfNames';
			$entry['member'] = "";
		}
		if (!OcHelper::isTestingOnReva()) {
			$entry['objectclass'][] = 'openCloudObject';
			$entry['openCloudUUID'] = WebDavHelper::generateUUIDv4();
		}

		try {
			$this->ldap->add($newDN, $entry);
		} catch (LdapException $e) {
			if (\str_contains($e->getMessage(), "Already exists")) {
				$this->ldap->delete(
					"cn=" . ldap_escape($group, "", LDAP_ESCAPE_DN)
					. ",ou=" . $this->ldapGroupsOU . "," . $this->ldapBaseDN,
				);
				$this->ldap->add($newDN, $entry);
			}
		}
		$this->ldapCreatedGroups[] = $group;
	}

	/**
	 * deletes LDAP users|groups created during test
	 *
	 * @return void
	 * @throws Exception
	 */
	public function deleteLdapUsersAndGroups(): void {
		foreach ($this->ldapCreatedUsers as $user) {
			$this->ldap->delete(
				"uid=" . ldap_escape($user, "", LDAP_ESCAPE_DN) . ",ou=" . $this->ldapUsersOU . "," . $this->ldapBaseDN,
			);
			$this->rememberThatUserIsNotExpectedToExist($user);
		}
		foreach ($this->ldapCreatedGroups as $group) {
			$this->ldap->delete(
				"cn=" . ldap_escape($group, "", LDAP_ESCAPE_DN)
				. ",ou=" . $this->ldapGroupsOU . "," . $this->ldapBaseDN,
			);
			$this->rememberThatGroupIsNotExpectedToExist($group);
		}
		if (!$this->skipImportLdif) {
			//delete all created ldap users
			$this->ldap->delete(
				"ou=" . $this->ldapUsersOU . "," . $this->ldapBaseDN,
				true
			);
			//delete all created ldap groups
			$this->ldap->delete(
				"ou=" . $this->ldapGroupsOU . "," . $this->ldapBaseDN,
				true
			);
		}
	}

	/**
	 * Creates multiple users
	 *
	 * This function will allow us to send user creation requests in parallel.
	 * This will be faster in comparison to waiting for each request to complete before sending another request.
	 *
	 * @param TableNode $table
	 * @param bool $useDefault
	 * @param bool $initialize
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function usersHaveBeenCreated(
		TableNode $table,
		bool $useDefault = true,
		bool $initialize = true
	) {
		$this->verifyTableNodeColumns($table, ['username'], ['displayname', 'email', 'password']);
		$table = $table->getColumnsHash();
		$users = $this->buildUsersAttributesArray($useDefault, $table);

		foreach ($users as $userAttributes) {
			$userName = $userAttributes['userid'];
			$password = $userAttributes['password'];
			$displayName = $userAttributes['displayName'];
			$email = $userAttributes['email'];

			if ($this->isTestingWithLdap()) {
				try {
					$this->createLdapUser($userAttributes);
				} catch (LdapException $exception) {
					throw new Exception(
						__METHOD__ . " cannot create a LDAP user with provided data. Error: $exception"
					);
				}
			} else {
				// Use the same logic as userHasBeenCreated for email generation
				if ($email === null) {
					$email = $this->getEmailAddressForUser($userName);
					if ($email === null) {
						// escape @ & space if present in userId
						$email = \str_replace(["@", " "], "", $userName) . '@opencloud.eu';
					}
				}

				$userName = $this->getActualUsername($userName);
				$userName = \trim($userName);

				$response = GraphHelper::createUser(
					$this->getBaseUrl(),
					$this->getStepLineRef(),
					$this->getAdminUsername(),
					$this->getAdminPassword(),
					$userName,
					$password,
					$email,
					$displayName,
				);

				Assert::assertEquals(
					201,
					$response->getStatusCode(),
					__METHOD__ . " cannot create user '$userName'.\nResponse:" .
					json_encode($this->getJsonDecodedResponse($response))
				);

				$userId = $this->getJsonDecodedResponse($response)['id'];
			}

			$this->addUserToCreatedUsersList($userName, $password, $displayName, $email, $userId ?? null);

			if ($initialize) {
				$this->initializeUser($userName, $password);
			}
		}
	}

	/**
	 * @param string $username
	 * @param string|null $password
	 *
	 * @return void
	 */
	public function resetUserPasswordAsAdminUsingTheProvisioningApi(string $username, ?string $password): void {
		$this->userResetUserPasswordUsingProvisioningApi(
			$this->getAdminUsername(),
			$username,
			$password
		);
	}

	/**
	 * @param string|null $user
	 * @param string|null $username
	 * @param string|null $password
	 *
	 * @return void
	 */
	public function userResetUserPasswordUsingProvisioningApi(
		?string $user,
		?string $username,
		?string $password
	): void {
		$targetUsername = $this->getActualUsername($username);
		$password = $this->getActualPassword($password);
		$this->userTriesToResetUserPasswordUsingTheProvisioningApi(
			$user,
			$targetUsername,
			$password
		);
		$this->rememberUserPassword($targetUsername, $password);
	}

	/**
	 * @param string|null $user
	 * @param string|null $username
	 * @param string|null $password
	 *
	 * @return void
	 */
	public function userTriesToResetUserPasswordUsingTheProvisioningApi(
		?string $user,
		?string $username,
		?string $password
	): void {
		$password = $this->getActualPassword($password);
		$bodyTable = new TableNode([['key', 'password'], ['value', $password]]);
		$this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			"PUT",
			"/cloud/users/$username",
			$bodyTable
		);
	}

	/**
	 * @When /^the administrator deletes user "([^"]*)" using the provisioning API$/
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theAdminDeletesUserUsingTheProvisioningApi(string $user): void {
		$user = $this->getActualUsername($user);
		$this->setResponse($this->deleteUser($user));
		$this->pushToLastHttpStatusCodesArray();
	}

	/**
	 * @Then /^user "([^"]*)" should exist$/
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws JsonException
	 */
	public function userShouldExist(string $user): void {
		Assert::assertTrue(
			$this->userExists($user),
			"User '$user' should exist but does not exist"
		);
	}

	/**
	 * @Then /^user "([^"]*)" should not exist$/
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws JsonException
	 */
	public function userShouldNotExist(string $user): void {
		$user = $this->getActualUsername($user);
		Assert::assertFalse(
			$this->userExists($user),
			"User '$user' should not exist but does exist"
		);
		$this->rememberThatUserIsNotExpectedToExist($user);
	}

	/**
	 * @Then /^group "([^"]*)" should exist$/
	 *
	 * @param string $group
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function groupShouldExist(string $group): void {
		Assert::assertTrue(
			$this->groupExists($group),
			"Group '$group' should exist but does not exist"
		);
	}

	/**
	 * @Then /^group "([^"]*)" should not exist$/
	 *
	 * @param string $group
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function groupShouldNotExist(string $group): void {
		Assert::assertFalse(
			$this->groupExists($group),
			"Group '$group' should not exist but does exist"
		);
	}

	/**
	 * @Then /^these groups should (not|)\s?exist:$/
	 * expects a table of groups with the heading "groupname"
	 *
	 * @param string $shouldOrNot (not|)
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theseGroupsShouldNotExist(string $shouldOrNot, TableNode $table): void {
		$should = ($shouldOrNot !== "not");
		$this->verifyTableNodeColumns($table, ['groupname']);
		if ($this->isTestingWithLdap()) {
			$groups = $this->getArrayOfGroupsResponded($this->getAllGroups());
			foreach ($table as $row) {
				if (\in_array($row['groupname'], $groups, true) !== $should) {
					throw new Exception(
						"group '" . $row['groupname'] .
						"' does" . ($should ? " not" : "") .
						" exist but should" . ($should ? "" : " not")
					);
				}
			}
		} else {
			$this->graphContext->theseGroupsShouldNotExist($shouldOrNot, $table);
		}
	}

	/**
	 * @Given /^user "([^"]*)" has been deleted$/
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userHasBeenDeleted(string $user): void {
		$user = $this->getActualUsername($user);
		if ($this->isTestingWithLdap() && \in_array($user, $this->ldapCreatedUsers)) {
			$this->deleteLdapUser($user);
		} else {
			$response = $this->deleteUser($user);
			$this->theHTTPStatusCodeShouldBe(204, "", $response);
			WebDavHelper::removeSpaceIdReferenceForUser($user);
		}
		$this->rememberThatUserIsNotExpectedToExist($user);
	}

	/**
	 * get all the existing groups
	 *
	 * @return ResponseInterface
	 */
	public function getAllGroups(): ResponseInterface {
		$fullUrl = $this->getBaseUrl() . "/ocs/v$this->ocsApiVersion.php/cloud/groups";
		return HttpRequestHelper::get(
			$fullUrl,
			$this->getStepLineRef(),
			$this->getAdminUsername(),
			$this->getAdminPassword()
		);
	}

	/**
	 * @param string $user
	 * @param string $otherUser
	 *
	 * @return void
	 */
	public function userGetsAllTheGroupsOfUser(string $user, string $otherUser): void {
		$actualOtherUser = $this->getActualUsername($otherUser);
		$fullUrl = $this->getBaseUrl() . "/ocs/v$this->ocsApiVersion.php/cloud/users/$actualOtherUser/groups";
		$actualUser = $this->getActualUsername($user);
		$actualPassword = $this->getUserPassword($actualUser);
		$this->response = HttpRequestHelper::get(
			$fullUrl,
			$this->getStepLineRef(),
			$actualUser,
			$actualPassword
		);
	}

	/**
	 * @When user :user gets the list of all users using the provisioning API
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userGetsTheListOfAllUsersUsingTheProvisioningApi(string $user): void {
		$this->featureContext->setResponse(
			$this->userGetsTheListOfAllUsers($user)
		);
	}

	/**
	 * @param string $user
	 *
	 * @return ResponseInterface
	 */
	public function userGetsTheListOfAllUsers(string $user): ResponseInterface {
		$fullUrl = $this->getBaseUrl() . "/ocs/v$this->ocsApiVersion.php/cloud/users";
		$actualUser = $this->getActualUsername($user);
		$actualPassword = $this->getUserPassword($actualUser);
		return HttpRequestHelper::get(
			$fullUrl,
			$this->getStepLineRef(),
			$actualUser,
			$actualPassword
		);
	}

	/**
	 * Make a request about the user. That will force the server to fully
	 * initialize the user, including their skeleton files.
	 *
	 * @param string $user
	 * @param string $password
	 *
	 * @return void
	 */
	public function initializeUser(string $user, string $password): void {
		$url = $this->getBaseUrl() . "/graph/v1.0/users/$user";

		if (OcHelper::isTestingOnReva()) {
			$url = $this->getBaseUrl()
				. "/ocs/v$this->ocsApiVersion.php/cloud/users/$user";
		}
		if ($password !== '') {
			HttpRequestHelper::get(
				$url,
				$this->getStepLineRef(),
				$user,
				$password
			);
		}
	}

	/**
	 * adds a user to the list of users that were created during test runs
	 * makes it possible to use this list in other test steps
	 * or to delete them at the end of the test
	 *
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $displayName
	 * @param string|null $email
	 * @param string|null $userId only set for the users created using the Graph API
	 * @param bool $shouldExist
	 *
	 * @return void
	 * @throws JsonException
	 */
	public function addUserToCreatedUsersList(
		?string $user,
		?string $password,
		?string $displayName = null,
		?string $email = null,
		?string $userId = null,
		bool $shouldExist = true
	): void {
		$user = $this->getActualUsername($user);
		$normalizedUsername = $this->normalizeUsername($user);
		$userData = [
			"password" => $password,
			"displayname" => $displayName,
			"email" => $email,
			"shouldExist" => $shouldExist,
			"actualUsername" => $user,
			"id" => $userId
		];

		if ($this->currentServer === 'LOCAL') {
			// Only remember this user creation if it was expected to have been successful
			// or the user has not been processed before. Some tests create a user the
			// first time (successfully) and then purposely try to create the user again.
			// The 2nd user creation is expected to fail, and in that case we want to
			// still remember the details of the first user creation.
			if ($shouldExist || !\array_key_exists($normalizedUsername, $this->createdUsers)) {
				$this->createdUsers[$normalizedUsername] = $userData;
			}
		} elseif ($this->currentServer === 'REMOTE') {
			// See comment above about the LOCAL case. The logic is the same for the remote case.
			if ($shouldExist || !\array_key_exists($normalizedUsername, $this->createdRemoteUsers)) {
				$this->createdRemoteUsers[$normalizedUsername] = $userData;
				$this->createdUsers[$normalizedUsername] = $userData;
			}
		}
	}

	/**
	 * remember the password of a user that already exists so that you can use
	 * ordinary test steps after changing their password.
	 *
	 * @param string $user
	 * @param string $password
	 *
	 * @return void
	 */
	public function rememberUserPassword(
		string $user,
		string $password
	): void {
		$normalizedUsername = $this->normalizeUsername($user);
		if ($this->currentServer === 'LOCAL') {
			if (\array_key_exists($normalizedUsername, $this->createdUsers)) {
				$this->createdUsers[$normalizedUsername]['password'] = $password;
			}
		} elseif ($this->currentServer === 'REMOTE') {
			if (\array_key_exists($normalizedUsername, $this->createdRemoteUsers)) {
				$this->createdRemoteUsers[$user]['password'] = $password;
			}
		}
	}

	/**
	 * @param string $oldUserName
	 * @param string $newUserName
	 *
	 * @return void
	 */
	public function updateUsernameInCreatedUserList(string $oldUserName, string $newUserName): void {
		$normalizedUsername = $this->normalizeUsername($oldUserName);
		$normalizeNewUserName = $this->normalizeUsername($newUserName);
		if (\array_key_exists($normalizedUsername, $this->createdUsers)) {
			foreach ($this->createdUsers as $createdUser) {
				if ($createdUser['actualUsername'] === $oldUserName) {
					$this->createdUsers[$normalizeNewUserName] = $this->createdUsers[$normalizedUsername];
					$this->createdUsers[$normalizeNewUserName]['actualUsername'] = $newUserName;
					unset($this->createdUsers[$normalizedUsername]);
				}
			}
		}
	}

	/**
	 * Remembers that a user from the list of users that were created during
	 * test runs is no longer expected to exist. Useful if a user was created
	 * during the setup phase but was deleted in a test run. We don't expect
	 * this user to exist in the tear-down phase, so remember that fact.
	 *
	 * @param string $user
	 *
	 * @return void
	 */
	public function rememberThatUserIsNotExpectedToExist(string $user): void {
		$user = $this->getActualUsername($user);
		$normalizedUsername = $this->normalizeUsername($user);
		if (\array_key_exists($normalizedUsername, $this->createdUsers)) {
			$this->createdUsers[$normalizedUsername]['shouldExist'] = false;
			$this->createdUsers[$normalizedUsername]['possibleToDelete'] = false;
		}
	}

	/**
	 * creates a single user
	 *
	 * @param array $userData
	 * @param string|null $byUser
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userHasBeenCreated(
		array $userData,
		string $byUser = null
	): void {
		$userId = null;

		$user = $userData["userName"];
		$displayName = $userData["displayName"] ?? null;
		$email = $userData["email"] ?? null;
		$password = $userData["password"] ?? null;

		if ($password === null) {
			$password = $this->getPasswordForUser($user);
		}

		if ($displayName === null) {
			$displayName = $this->getDisplayNameForUser($user);
			if ($displayName === null) {
				$displayName = $this->getDisplayNameForUser('regularuser');
			}
		}

		if ($email === null) {
			$email = $this->getEmailAddressForUser($user);

			if ($email === null) {
				// escape @ & space if present in userId
				$email = \str_replace(["@", " "], "", $user) . '@opencloud.eu';
			}
		}
		$user = $this->getActualUsername($user);
		$user = \trim($user);

		if ($this->isTestingWithLdap()) {
			$setting["userid"] = $user;
			$setting["displayName"] = $displayName;
			$setting["password"] = $password;
			$setting["email"] = $email;
			try {
				$this->createLdapUser($setting);
			} catch (LdapException $exception) {
				throw new Exception(
					__METHOD__ . " cannot create a LDAP user with provided data. Error: $exception"
				);
			}
		} else {
			$reqUser = $byUser ? $this->getActualUsername($byUser) : $this->getAdminUsername();
			$response = GraphHelper::createUser(
				$this->getBaseUrl(),
				$this->getStepLineRef(),
				$reqUser,
				$this->getPasswordForUser($reqUser),
				$user,
				$password,
				$email,
				$displayName,
			);
			Assert::assertEquals(
				201,
				$response->getStatusCode(),
				__METHOD__ . " cannot create user '$user'.\nResponse:" .
				json_encode($this->getJsonDecodedResponse($response))
			);
			$userId = $this->getJsonDecodedResponse($response)['id'];
		}

		$this->addUserToCreatedUsersList($user, $password, $displayName, $email, $userId);
		$this->initializeUser($user, $password);
	}

	/**
	 * @When the administrator removes user :user from group :group using the provisioning API
	 *
	 * @param string $user
	 * @param string $group
	 *
	 * @return void
	 * @throws Exception
	 */
	public function adminRemovesUserFromGroupUsingTheProvisioningApi(string $user, string $group): void {
		$user = $this->getActualUsername($user);
		if (OcHelper::isTestingOnReva()) {
			$this->response = UserHelper::removeUserFromGroup(
				$this->getBaseUrl(),
				$user,
				$group,
				$this->getAdminUsername(),
				$this->getAdminPassword(),
				$this->getStepLineRef(),
				$this->ocsApiVersion
			);
		} else {
			$this->setResponse(
				$this->graphContext->removeUserFromGroup(
					$group,
					$user
				)
			);
		}

		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @Then /^the extra groups returned by the API should be$/
	 *
	 * @param TableNode $groupsList
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theExtraGroupsShouldBe(TableNode $groupsList): void {
		$this->verifyTableNodeColumnsCount($groupsList, 1);
		$groups = $groupsList->getRows();
		$groupsSimplified = $this->simplifyArray($groups);
		if ($this->isTestingWithLdap()) {
			$expectedGroups = \array_merge($this->startingGroups, $groupsSimplified);
			$respondedArray = $this->getArrayOfGroupsResponded($this->response);
			Assert::assertEqualsCanonicalizing(
				$expectedGroups,
				$respondedArray,
				__METHOD__
				. " Provided groups do not match the groups returned in the response."
			);
		} else {
			$this->graphContext->theseGroupsShouldBeInTheResponse($groupsSimplified);
		}
	}

	/**
	 * Try to delete the group, catching anything bad that might happen.
	 * Use this method only in places where you want to try as best you
	 * can to delete the group, but do not want to error if there is a problem.
	 *
	 * @param string $group
	 *
	 * @return void
	 * @throws Exception
	 */
	public function cleanupGroup(string $group): void {
		try {
			if ($this->isTestingWithLdap()) {
				$this->deleteLdapGroup($group);
			} else {
				$response = $this->graphContext->deleteGroupWithName($group);
				$this->theHTTPStatusCodeShouldBe(204, "", $response);
			}
		} catch (Exception $e) {
			\error_log(
				"INFORMATION: There was an unexpected problem trying to delete group " .
				"'$group' message '" . $e->getMessage() . "'"
			);
		}

		if ($this->theGroupShouldBeAbleToBeDeleted($group)
			&& $this->groupExists($group)
		) {
			\error_log(
				"INFORMATION: tried to delete group '$group'" .
				" at the end of the scenario but it seems to still exist. " .
				"There might be problems with later scenarios."
			);
		}
	}

	/**
	 * @param string $user
	 *
	 * @return bool
	 * @throws JsonException
	 */
	public function userExists(string $user): bool {
		$path = (!OcHelper::isTestingOnReva())
			? "/graph/v1.0"
			: "/ocs/v2.php/cloud";
		$fullUrl = $this->getBaseUrl() . $path . "/users/$user";

		if (OcHelper::isTestingOnReva()) {
			$requestingUser = $this->getActualUsername($user);
			$requestingPassword = $this->getPasswordForUser($user);
		} else {
			$requestingUser = $this->getAdminUsername();
			$requestingPassword = $this->getAdminPassword();
		}

		$response = HttpRequestHelper::get(
			$fullUrl,
			$this->getStepLineRef(),
			$requestingUser,
			$requestingPassword
		);
		if ($response->getStatusCode() >= 400) {
			return false;
		}
		return true;
	}

	/**
	 * @Then /^user "([^"]*)" should belong to group "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $group
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldBelongToGroup(string $user, string $group): void {
		$user = $this->getActualUsername($user);
		if (OcHelper::isTestingOnReva()) {
			$this->userGetsAllTheGroupsOfUser($this->getAdminUsername(), $user);
			$respondedArray = $this->getArrayOfGroupsResponded($this->response);
			\sort($respondedArray);
			Assert::assertContains(
				$group,
				$respondedArray,
				__METHOD__ . " Group '$group' does not exist in '"
				. \implode(', ', $respondedArray)
				. "'"
			);
			Assert::assertEquals(
				200,
				$this->response->getStatusCode(),
				__METHOD__
				. " Expected status code is '200' but got '"
				. $this->response->getStatusCode()
				. "'"
			);
		} else {
			$this->graphContext->userShouldBeMemberInGroupUsingTheGraphApi(
				$user,
				$group
			);
		}
	}

	/**
	 * @Then the following users should not belong to the following groups
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theTheFollowingUserShouldNotBelongToTheFollowingGroup(TableNode $table): void {
		$this->verifyTableNodeColumns($table, ["username", "groupname"]);
		$rows = $table->getHash();
		foreach ($rows as $row) {
			$user = $this->getActualUsername($row["username"]);
			$group = $row["groupname"];
			if (OcHelper::isTestingOnReva()) {
				$fullUrl = $this->getBaseUrl() . "/ocs/v2.php/cloud/users/$user/groups";
				$response = HttpRequestHelper::get(
					$fullUrl,
					$this->getStepLineRef(),
					$this->getAdminUsername(),
					$this->getAdminPassword()
				);
				$respondedArray = $this->getArrayOfGroupsResponded($response);
				\sort($respondedArray);
				Assert::assertNotContains($group, $respondedArray);
				Assert::assertEquals(
					200,
					$response->getStatusCode()
				);
			} else {
				$this->graphContext->userShouldNotBeMemberInGroupUsingTheGraphApi($user, $group);
			}
		}
	}

	/**
	 * @Then group :group should not contain user :username
	 *
	 * @param string $group
	 * @param string $username
	 *
	 * @return void
	 */
	public function groupShouldNotContainUser(string $group, string $username): void {
		$username = $this->getActualUsername($username);
		$fullUrl = $this->getBaseUrl() . "/ocs/v2.php/cloud/groups/$group";
		$response = HttpRequestHelper::get(
			$fullUrl,
			$this->getStepLineRef(),
			$this->getAdminUsername(),
			$this->getAdminPassword()
		);
		Assert::assertNotContains($username, $this->getArrayOfUsersResponded($response));
	}

	/**
	 * @When /^the administrator adds user "([^"]*)" to group "([^"]*)" using the provisioning API$/
	 *
	 * @param string $user
	 * @param string $group
	 *
	 * @return void
	 * @throws Exception
	 */
	public function adminAddsUserToGroupUsingTheProvisioningApi(string $user, string $group): void {
		$response = $this->graphContext->addUserToGroup($group, $user);
		$this->setResponse($response);
	}

	/**
	 * @Given /^user "([^"]*)" has been added to group "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $group
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userHasBeenAddedToGroup(string $user, string $group): void {
		$user = $this->getActualUsername($user);
		if ($this->isTestingWithLdap()) {
			try {
				$this->addUserToLdapGroup(
					$user,
					$group
				);
			} catch (LdapException $exception) {
				throw new Exception(
					"User $user cannot be added to $group Error: $exception"
				);
			}
		} else {
			$response = $this->graphContext->addUserToGroup($group, $user);
			$this->theHTTPStatusCodeShouldBe(204, '', $response);
		}
	}

	/**
	 * @Given the following users have been added to the following groups
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theFollowingUserHaveBeenAddedToTheFollowingGroup(TableNode $table): void {
		$this->verifyTableNodeColumns($table, ['username', 'groupname']);
		foreach ($table as $row) {
			$user = $row['username'];
			$group = $row['groupname'];
			$user = $this->getActualUsername($user);
			if ($this->isTestingWithLdap()) {
				try {
					$this->addUserToLdapGroup(
						$user,
						$group
					);
				} catch (LdapException $exception) {
					throw new Exception(
						"User $user cannot be added to $group Error: $exception"
					);
				}
			} else {
				$response = $this->graphContext->addUserToGroup($group, $user);
				$this->theHTTPStatusCodeShouldBe(204, '', $response);
			}
		}
	}

	/**
	 * @param string $group
	 * @param bool $shouldExist - true if the group should exist
	 * @param bool $possibleToDelete - true if it is possible to delete the group
	 * @param string|null $id - id of the group, only required for the groups created using the Graph API
	 *
	 * @return void
	 */
	public function addGroupToCreatedGroupsList(
		string $group,
		bool $shouldExist = true,
		bool $possibleToDelete = true,
		?string $id = null
	): void {
		$groupData = [
			"shouldExist" => $shouldExist,
			"possibleToDelete" => $possibleToDelete
		];
		if ($id !== null) {
			$groupData["id"] = $id;
		}

		if ($this->currentServer === 'LOCAL') {
			$this->createdGroups[$group] = $groupData;
		} elseif ($this->currentServer === 'REMOTE') {
			$this->createdRemoteGroups[$group] = $groupData;
		}
	}

	/**
	 * Remembers that a group from the list of groups that were created during
	 * test runs is no longer expected to exist. Useful if a group was created
	 * during the setup phase but was deleted in a test run. We don't expect
	 * this group to exist in the tear-down phase, so remember that fact.
	 *
	 * @param string $group
	 *
	 * @return void
	 */
	public function rememberThatGroupIsNotExpectedToExist(string $group): void {
		if (\array_key_exists($group, $this->createdGroups)) {
			$this->createdGroups[$group]['shouldExist'] = false;
			$this->createdGroups[$group]['possibleToDelete'] = false;
		}
	}

	/**
	 * @Given /^group "([^"]*)" has been created$/
	 *
	 * @param string $group
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function groupHasBeenCreated(string $group): void {
		$this->createTheGroup($group);
		Assert::assertTrue(
			$this->groupExists($group),
			"Group '$group' should exist but does not exist"
		);
	}

	/**
	 * @Given these groups have been created:
	 * expects a table of groups with the heading "groupname"
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theseGroupsHaveBeenCreated(TableNode $table): void {
		$this->verifyTableNodeColumns($table, ['groupname'], ['comment']);
		foreach ($table as $row) {
			$this->createTheGroup($row['groupname']);
		}
	}

	/**
	 * creates a single group
	 *
	 * @param string $group
	 * @param string|null $method how to create the group api|occ
	 *
	 * @return void
	 * @throws Exception
	 */
	public function createTheGroup(string $group, ?string $method = null): void {
		//guess yourself
		if ($method === null) {
			if ($this->isTestingWithLdap()) {
				$method = "ldap";
			} else {
				$method = "graph";
			}
		}
		$group = \trim($group);
		$method = \trim(\strtolower($method));
		$groupCanBeDeleted = false;
		$groupId = null;
		switch ($method) {
			case "ldap":
				try {
					$this->createLdapGroup($group);
				} catch (LdapException $e) {
					throw new Exception(
						"could not create group '$group'. Error: $e"
					);
				}
				break;
			case "graph":
				$newGroup = $this->graphContext->createGroup($group);
				if ($newGroup->getStatusCode() === 201) {
					$newGroup = $this->getJsonDecodedResponse($newGroup);
				}
				$groupCanBeDeleted = true;
				$groupId = $newGroup["id"];
				break;
			default:
				throw new InvalidArgumentException(
					"Invalid method to create group '$group'"
				);
		}

		$this->addGroupToCreatedGroupsList($group, true, $groupCanBeDeleted, $groupId);
	}

	/**
	 * @param string $attribute
	 * @param string $entry
	 * @param string $value
	 * @param bool $append
	 *
	 * @return void
	 * @throws Exception
	 */
	public function setTheLdapAttributeOfTheEntryTo(
		string $attribute,
		string $entry,
		string $value,
		bool $append = false
	): void {
		$ldapEntry = $this->ldap->getEntry($entry . "," . $this->ldapBaseDN);
		Laminas\Ldap\Attribute::setAttribute($ldapEntry, $attribute, $value, $append);
		$this->ldap->update($entry . "," . $this->ldapBaseDN, $ldapEntry);
	}

	/**
	 * @param string $user
	 * @param string $group
	 * @param string|null $ou
	 *
	 * @return void
	 * @throws Exception
	 */
	public function addUserToLdapGroup(string $user, string $group, ?string $ou = null): void {
		if ($ou === null) {
			$ou = $this->getLdapGroupsOU();
		}
		if ($this->ldapGroupSchema == "rfc2307") {
			$memberAttr = "memberUID";
			$memberValue = "$user";
		} else {
			$memberAttr = "member";
			$userbase = "ou=" . $this->getLdapUsersOU() . "," . $this->ldapBaseDN;
			$memberValue = "uid=$user" . "," . "$userbase";
		}
		$this->setTheLdapAttributeOfTheEntryTo(
			$memberAttr,
			"cn=$group,ou=$ou",
			$memberValue,
			true
		);
	}

	/**
	 * @param string $value
	 * @param string $attribute
	 * @param string $entry
	 *
	 * @return void
	 */
	public function deleteValueFromLdapAttribute(string $value, string $attribute, string $entry): void {
		$this->ldap->deleteAttributes(
			$entry . "," . $this->ldapBaseDN,
			[$attribute => [$value]]
		);
	}

	/**
	 * @param string $user
	 * @param string $group
	 * @param string|null $ou
	 *
	 * @return void
	 * @throws Exception
	 */
	public function removeUserFromLdapGroup(string $user, string $group, ?string $ou = null): void {
		if ($ou === null) {
			$ou = $this->getLdapGroupsOU();
		}
		if ($this->ldapGroupSchema == "rfc2307") {
			$memberAttr = "memberUID";
			$memberValue = "$user";
		} else {
			$memberAttr = "member";
			$userbase = "ou=" . $this->getLdapUsersOU() . "," . $this->ldapBaseDN;
			$memberValue = "uid=$user" . "," . "$userbase";
		}
		$this->deleteValueFromLdapAttribute(
			$memberValue,
			$memberAttr,
			"cn=$group,ou=$ou"
		);
	}

	/**
	 * @param string $entry
	 *
	 * @return void
	 * @throws Exception
	 */
	public function deleteTheLdapEntry(string $entry): void {
		$this->ldap->delete($entry . "," . $this->ldapBaseDN);
	}

	/**
	 * @param string $group
	 * @param string|null $ou
	 *
	 * @return void
	 * @throws LdapException
	 * @throws Exception
	 */
	public function deleteLdapGroup(string $group, ?string $ou = null): void {
		if ($ou === null) {
			$ou = $this->getLdapGroupsOU();
		}
		$this->deleteTheLdapEntry("cn=$group,ou=$ou");
		$key = \array_search($group, $this->ldapCreatedGroups);
		if ($key !== false) {
			unset($this->ldapCreatedGroups[$key]);
		}
		$this->rememberThatGroupIsNotExpectedToExist($group);
	}

	/**
	 * @param string|null $username
	 * @param string|null $ou
	 *
	 * @return void
	 * @throws Exception
	 */
	public function deleteLdapUser(?string $username, ?string $ou = null): void {
		if (!\in_array($username, $this->ldapCreatedUsers)) {
			throw new Error(
				"User " . $username . " was not created using Ldap and does not exist as an Ldap User"
			);
		}
		if ($ou === null) {
			$ou = $this->getLdapUsersOU();
		}
		$entry = "uid=$username,ou=$ou";
		$this->deleteTheLdapEntry($entry);
		$key = \array_search($username, $this->ldapCreatedUsers);
		if ($key !== false) {
			unset($this->ldapCreatedUsers[$key]);
		}
		$this->rememberThatUserIsNotExpectedToExist($username);
	}

	/**
	 * @Given /^user "([^"]*)" has been disabled$/
	 *
	 * @param string|null $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public function adminHasDisabledUserUsingTheProvisioningApi(?string $user): void {
		$user = $this->getActualUsername($user);
		if (OcHelper::isTestingOnReva()) {
			$response = $this->disableOrEnableUser($this->getAdminUsername(), $user, 'disable');
		} else {
			$response = $this->graphContext->editUserUsingTheGraphApi(
				$this->getAdminUsername(),
				$user,
				null,
				null,
				null,
				null,
				false
			);
		}
		Assert::assertEquals(
			200,
			$response->getStatusCode(),
			__METHOD__
			. " Expected status code is 200 but received " . $response->getStatusCode()
			. "\nResponse body: " . $response->getBody(),
		);
	}

	/**
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public function deleteUser(string $user): ResponseInterface {
		// Always try to delete the user
		if (OcHelper::isTestingOnReva()) {
			$response = UserHelper::deleteUser(
				$this->getBaseUrl(),
				$user,
				$this->getAdminUsername(),
				$this->getAdminPassword(),
				$this->getStepLineRef(),
				$this->ocsApiVersion
			);
		} else {
			// users can be deleted using the username in the GraphApi too
			$response = $this->graphContext->adminDeletesUserUsingTheGraphApi($user);
		}
		return $response;
	}

	/**
	 * @Given /^group "([^"]*)" has been deleted$/
	 *
	 * @param string $group
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function groupHasBeenDeleted(string $group): void {
		if ($this->isTestingWithLdap()) {
			$this->deleteLdapGroup($group);
		} else {
			$response = $this->graphContext->deleteGroupWithName($group);
			$this->theHTTPStatusCodeShouldBe(204, "", $response);
		}
		$this->rememberThatGroupIsNotExpectedToExist($group);
		Assert::assertFalse(
			$this->groupExists($group),
			"Group '$group' should not exist but does exist"
		);
	}

	/**
	 * @param string $group
	 *
	 * @return bool
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function groupExists(string $group): bool {
		if ($this->isTestingWithLdap()) {
			$baseDN = $this->getLdapBaseDN();
			$newDN = 'cn=' . $group . ',ou=' . $this->ldapGroupsOU . ',' . $baseDN;
			if ($this->ldap->getEntry($newDN) !== null) {
				return true;
			}
			return false;
		}
		$group = \rawurlencode($group);
		$fullUrl = $this->getBaseUrl() . "/graph/v1.0/groups/$group";
		$this->response = HttpRequestHelper::get(
			$fullUrl,
			$this->getStepLineRef(),
			$this->getAdminUsername(),
			$this->getAdminPassword()
		);
		if ($this->response->getStatusCode() >= 400) {
			return false;
		}
		return true;
	}

	/**
	 * @Given user :user has been removed from group :group
	 *
	 * @param string $user
	 * @param string $group
	 *
	 * @return void
	 * @throws Exception
	 */
	public function adminHasRemovedUserFromGroup(string $user, string $group): void {
		$user = $this->getActualUsername($user);
		if ($this->isTestingWithLdap()
			&& !$this->isLocalAdminGroup($group)
			&& \in_array($group, $this->ldapCreatedGroups)
		) {
			$this->removeUserFromLdapGroup($user, $group);
		} else {
			$response = $this->graphContext->removeUserFromGroup($group, $user);
			$this->TheHTTPStatusCodeShouldBe(204, '', $response);
		}

		if (OcHelper::isTestingOnReva()) {
			$fullUrl = $this->getBaseUrl() . "/ocs/v2.php/cloud/users/$user/groups";
			$response = HttpRequestHelper::get(
				$fullUrl,
				$this->getStepLineRef(),
				$this->getAdminUsername(),
				$this->getAdminPassword()
			);
			$respondedArray = $this->getArrayOfGroupsResponded($response);
			\sort($respondedArray);
			Assert::assertNotContains($group, $respondedArray);
			Assert::assertEquals(
				200,
				$response->getStatusCode()
			);
		} else {
			$this->graphContext->userShouldNotBeMemberInGroupUsingTheGraphApi($user, $group);
		}
	}

	/**
	 * @Then /^the users returned by the API should be$/
	 *
	 * @param TableNode $usersList
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theUsersShouldBe(TableNode $usersList): void {
		$this->verifyTableNodeColumnsCount($usersList, 1);
		$users = $usersList->getRows();
		$usersSimplified = \array_map(
			function ($user) {
				return $this->getActualUsername($user);
			},
			$this->simplifyArray($users)
		);
		if ($this->isTestingWithLdap()) {
			$respondedArray = $this->getArrayOfUsersResponded($this->response);
			Assert::assertEqualsCanonicalizing(
				$usersSimplified,
				$respondedArray,
				__METHOD__
				. " Provided users do not match the users returned in the response."
			);
		} else {
			$this->graphContext->theseUsersShouldBeInTheResponse($usersSimplified);
		}
	}

	/**
	 * Parses the xml answer to get the array of users returned.
	 *
	 * @param ResponseInterface $resp
	 *
	 * @return array
	 * @throws Exception
	 */
	public function getArrayOfUsersResponded(ResponseInterface $resp): array {
		$listCheckedElements = HttpRequestHelper::getResponseXml($resp, __METHOD__)->data[0]->users[0]->element;
		return \json_decode(\json_encode($listCheckedElements), true);
	}

	/**
	 * Parses the xml answer to get the array of groups returned.
	 *
	 * @param ResponseInterface $resp
	 *
	 * @return array
	 * @throws Exception
	 */
	public function getArrayOfGroupsResponded(ResponseInterface $resp): array {
		$listCheckedElements
			= HttpRequestHelper::getResponseXml($resp, __METHOD__)->data[0]->groups[0]->element;
		return \json_decode(\json_encode($listCheckedElements), true);
	}

	/**
	 * Parses the xml answer to get the array of apps returned.
	 *
	 * @param ResponseInterface $resp
	 *
	 * @return array
	 * @throws Exception
	 */
	public function getArrayOfAppsResponded(ResponseInterface $resp): array {
		$listCheckedElements = HttpRequestHelper::getResponseXml($resp, __METHOD__)->data[0]->apps[0]->element;
		return \json_decode(\json_encode($listCheckedElements), true);
	}

	/**
	 * @Then /^the API should not return any data$/
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theApiShouldNotReturnAnyData(): void {
		$responseData = HttpRequestHelper::getResponseXml($this->response, __METHOD__)->data[0];
		Assert::assertEmpty(
			$responseData,
			"Response data is not empty but it should be empty"
		);
	}

	/**
	 *
	 * @return void
	 * @throws Exception
	 */
	public function cleanupDatabaseUsers(): void {
		$previousServer = $this->currentServer;
		$this->usingServer('LOCAL');
		foreach ($this->createdUsers as $userData) {
			$user = $userData['actualUsername'];
			TokenHelper::clearUserTokens($user, $this->getBaseUrl());
			$this->deleteUser($user);
			$this->rememberThatUserIsNotExpectedToExist($user);
		}
		$this->usingServer('REMOTE');
		foreach ($this->createdRemoteUsers as $userData) {
			$user = $userData['actualUsername'];
			TokenHelper::clearUserTokens($user, $this->getBaseUrl());
			$this->deleteUser($user);
			$this->rememberThatUserIsNotExpectedToExist($user);
		}
		$this->usingServer($previousServer);
	}

	/**
	 *
	 * @return void
	 * @throws Exception
	 */
	public function cleanupDatabaseGroups(): void {
		$previousServer = $this->currentServer;
		$this->usingServer('LOCAL');
		foreach ($this->createdGroups as $group => $groupData) {
			if ($groupData["possibleToDelete"]) {
				if ($this->isTestingWithLdap()) {
					$this->cleanupGroup((string)$group);
				} else {
					$response = $this->graphContext->deleteGroupWithId($groupData['id']);
					$this->theHTTPStatusCodeShouldBe(204, "", $response);
				}
			}
		}
		$this->usingServer('REMOTE');
		foreach ($this->createdRemoteGroups as $remoteGroup => $groupData) {
			if ($groupData["possibleToDelete"]) {
				if ($this->isTestingWithLdap()) {
					$this->cleanupGroup((string)$remoteGroup);
				} else {
					$response = $this->graphContext->deleteGroupWithId($groupData['id']);
					$this->theHTTPStatusCodeShouldBe(204, "", $response);
				}
			}
		}
		$this->usingServer($previousServer);
	}

	/**
	 * @BeforeScenario @rememberGroupsThatExist
	 *
	 * @return void
	 * @throws Exception
	 */
	public function rememberGroupsThatExistAtTheStartOfTheScenario(): void {
		$this->startingGroups = $this->getArrayOfGroupsResponded($this->getAllGroups());
	}

	/**
	 * disable or enable user
	 *
	 * @param string $user
	 * @param string $otherUser
	 * @param string $action
	 *
	 * @return void
	 */
	public function disableOrEnableUser(string $user, string $otherUser, string $action): ResponseInterface {
		$actualUser = $this->getActualUsername($user);
		$actualPassword = $this->getPasswordForUser($actualUser);
		$actualOtherUser = $this->getActualUsername($otherUser);

		$fullUrl = $this->getBaseUrl()
			. "/ocs/v$this->ocsApiVersion.php/cloud/users/$actualOtherUser/$action";
		return HttpRequestHelper::put(
			$fullUrl,
			$this->getStepLineRef(),
			$actualUser,
			$actualPassword
		);
	}
}
