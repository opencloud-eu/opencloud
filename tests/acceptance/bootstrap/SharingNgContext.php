<?php declare(strict_types=1);
/**
 * @author Viktor Scharf <scharf.vi@gmail.com>
 *
 * @copyright Copyright (c) 2023, ownCloud GmbH
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

use Behat\Behat\Context\Context;
use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use Behat\Gherkin\Node\TableNode;
use GuzzleHttp\Exception\GuzzleException;
use PHPUnit\Framework\Assert;
use Psr\Http\Message\ResponseInterface;
use TestHelpers\GraphHelper;
use TestHelpers\WebDavHelper;
use TestHelpers\HttpRequestHelper;
use TestHelpers\BehatHelper;

require_once 'bootstrap.php';

/**
 * Acceptance test steps related to testing sharing ng features
 */
class SharingNgContext implements Context {
	private FeatureContext $featureContext;
	private SpacesContext $spacesContext;

	/**
	 * This will run before EVERY scenario.
	 * It will set the properties for this object.
	 *
	 * @BeforeScenario
	 *
	 * @param BeforeScenarioScope $scope
	 *
	 * @return void
	 */
	public function before(BeforeScenarioScope $scope): void {
		// Get the environment
		$environment = $scope->getEnvironment();
		// Get all the contexts you need in this context from here
		$this->featureContext = BehatHelper::getContext($scope, $environment, 'FeatureContext');
		$this->spacesContext = BehatHelper::getContext($scope, $environment, 'SpacesContext');
	}

	/**
	 * Create link share of item (resource) or drive (space) using drives.permissions endpoint
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function createLinkShare(string $user, TableNode $body): ResponseInterface {
		$bodyRows = $body->getRowsHash();
		$resource = $bodyRows['resource'] ?? "";

		$space = $this->spacesContext->getSpaceByName($user, $bodyRows['space']);
		$spaceId = $space['id'];

		if ($resource === '' && !\in_array($bodyRows['space'], ['Personal', 'Shares'])) {
			$itemId = $space['fileId'];
		} else {
			$itemId = $this->spacesContext->getResourceId($user, $bodyRows['space'], $resource);
		}

		$bodyRows['quickLink'] = $bodyRows['quickLink'] ?? false;
		$bodyRows['displayName'] = $bodyRows['displayName'] ?? null;
		$bodyRows['expirationDateTime'] = \array_key_exists('expirationDateTime', $bodyRows)
		? \date('Y-m-d', \strtotime($bodyRows['expirationDateTime'])) . 'T14:00:00.000Z' : null;
		$bodyRows['password'] = $bodyRows['password'] ?? null;
		$body = [
			'type' => $bodyRows['permissionsRole'],
			"@libre.graph.quickLink" => filter_var($bodyRows['quickLink'], FILTER_VALIDATE_BOOLEAN),
			'displayName' => $bodyRows['displayName'],
			'expirationDateTime' => $bodyRows['expirationDateTime'],
			'password' => $this->featureContext->getActualPassword($bodyRows['password'])
		];

		return GraphHelper::createLinkShare(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$spaceId,
			$itemId,
			\json_encode($body)
		);
	}

	/**
	 * Create link share of drive (space) using drives.root endpoint
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function createDriveLinkShare(string $user, TableNode $body): ResponseInterface {
		$bodyRows = $body->getRowsHash();
		$space = $bodyRows['space'];

		$spaceId = ($this->spacesContext->getSpaceByName($user, $space))["id"];
		$bodyRows['quickLink'] = $bodyRows['quickLink'] ?? false;
		$bodyRows['displayName'] = $bodyRows['displayName'] ?? null;
		$bodyRows['expirationDateTime'] = $bodyRows['expirationDateTime'] ?? null;
		$bodyRows['password'] = $bodyRows['password'] ?? null;
		$body = [
			'type' => $bodyRows['permissionsRole'],
			"@libre.graph.quickLink" => filter_var($bodyRows['quickLink'], FILTER_VALIDATE_BOOLEAN),
			'displayName' => $bodyRows['displayName'],
			'expirationDateTime' => $bodyRows['expirationDateTime'],
			'password' => $this->featureContext->getActualPassword($bodyRows['password'])
		];

		return GraphHelper::createDriveShareLink(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$spaceId,
			\json_encode($body)
		);
	}

	/**
	 * @param string $user
	 * @param string $fileOrFolder (file|folder)
	 * @param string $space
	 * @param string|null $resource
	 * @param string|null $query
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function getPermissionsList(
		string $user,
		string $fileOrFolder,
		string $space,
		?string $resource = '',
		?string $query = null
	): ResponseInterface {
		$spaceId = ($this->spacesContext->getSpaceByName($user, $space))["id"];

		if ($fileOrFolder === 'folder') {
			$itemId = $this->spacesContext->getResourceId($user, $space, $resource);
		} else {
			$itemId = $this->spacesContext->getFileId($user, $space, $resource);
		}

		return GraphHelper::getPermissionsList(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$spaceId,
			$itemId,
			$query
		);
	}

	/**
	 * @param string $user
	 * @param string $space
	 * @param string|null $query
	 * @param string|null $spaceOwner
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	private function getDrivePermissionsList(
		string $user,
		string $space,
		?string $query = null,
		?string $spaceOwner = null,
	): ResponseInterface {
		if ($spaceOwner) {
			$spaceId = ($this->spacesContext->getSpaceByName($spaceOwner, $space))["id"];
		} else {
			$spaceId = ($this->spacesContext->getSpaceByName($user, $space))["id"];
		}

		return GraphHelper::getDrivePermissionsList(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$spaceId,
			$query
		);
	}

	/**
	 * @When /^user "([^"]*)" gets permissions list for (folder|file) "([^"]*)" of the space "([^"]*)" using the Graph API$/
	 *
	 * @param string $user
	 * @param string $fileOrFolder   (file|folder)
	 * @param string $resource
	 * @param string $space
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userGetsPermissionsListForResourceOfTheSpaceUsingTheGraphAPI(
		string $user,
		string $fileOrFolder,
		string $resource,
		string $space
	): void {
		$this->featureContext->setResponse(
			$this->getPermissionsList($user, $fileOrFolder, $space, $resource)
		);
	}

	/**
	 * @When /^user "([^"]*)" lists the permissions of space "([^"]*)" using permissions endpoint of the Graph API$/
	 *
	 * @param string $user
	 * @param string $space
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userListsThePermissionsOfSpaceUsingTheGraphApi(string $user, string $space): void {
		$this->featureContext->setResponse(
			$this->getPermissionsList($user, 'folder', $space)
		);
	}

	/**
	 * @When /^user "([^"]*)" tries to list the permissions of space "([^"]*)" owned by "([^"]*)" using permissions endpoint of the Graph API$/
	 *
	 * @param string $user
	 * @param string $space
	 * @param string $spaceOwner
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userTriesToListThePermissionsOfSpaceUsingPermissionsEndpointOfTheGraphApi(
		string $user,
		string $space,
		string $spaceOwner
	): void {
		$spaceId = ($this->spacesContext->getSpaceByName($spaceOwner, $space))["id"];
		$itemId = $this->spacesContext->getResourceId($spaceOwner, $space, '');

		$this->featureContext->setResponse(
			GraphHelper::getPermissionsList(
				$this->featureContext->getBaseUrl(),
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$spaceId,
				$itemId
			)
		);
	}

	/**
	 * share the item (resource) or drive (space) using the drives.permissions endpoint
	 *
	 * @param string $user
	 * @param array $shareInfo
	 * @param string|null $fileId
	 * @param bool $federatedShare
	 *
	 * @return ResponseInterface
	 *
	 * @throws JsonException
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function sendShareInvitation(
		string $user,
		array $shareInfo,
		?string $fileId = null,
		bool $federatedShare = false
	): ResponseInterface {
		$space = $this->spacesContext->getSpaceByName($user, $shareInfo['space']);
		$spaceId = $space['id'];

		// $fileId is used for trying to share deleted files
		if ($fileId) {
			$itemId = $fileId;
		} else {
			$resource = $shareInfo['resource'] ?? '';

			// for a disabled and deleted space, resource id is not accessible, so get resource id from the saved response
			if ($resource === '' && !\in_array($shareInfo['space'], ['Personal', 'Shares'])) {
				$itemId = $space['fileId'];
			} else {
				$itemId = $this->spacesContext->getResourceId($user, $shareInfo['space'], $resource);
			}
		}

		$shareeIds = [];

		if (\array_key_exists('shareeId', $shareInfo)) {
			$shareeIds[] = $shareInfo['shareeId'];
			$shareTypes[] = $shareInfo['shareType'];
		} else {
			$sharees = array_map('trim', explode(',', $shareInfo['sharee']));
			$shareTypes = array_map('trim', explode(',', $shareInfo['shareType']));

			foreach ($sharees as $index => $sharee) {
				$shareType = $shareTypes[$index];
				$shareeId = "";
				if ($shareType === "user") {
					$shareeId = $this->featureContext->getAttributeOfCreatedUser($sharee, 'id');
					if ($federatedShare) {
						$federatedUser = $this->featureContext->ocmContext->getAcceptedUserByName(
							$user,
							$sharee
						);
						$shareeId = $federatedUser['user_id'] . "@" . $federatedUser['idp'];
					}
				} elseif ($shareType === "group") {
					$shareeId = $this->featureContext->getAttributeOfCreatedGroup($sharee, 'id');
				}
				// for non-existing group or user, generate random id
				$shareeIds[] = $shareeId ?: WebDavHelper::generateUUIDv4();
			}
		}

		$permissionsRole = $shareInfo['permissionsRole'] ?? null;
		$permissionsAction = $shareInfo['permissionsAction'] ?? null;
		$expirationDateTime = (\array_key_exists('expirationDateTime', $shareInfo))
			? \date(DATE_RFC3339, \strtotime($shareInfo['expirationDateTime'])) : null;

		$response = GraphHelper::sendSharingInvitation(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$spaceId,
			$itemId,
			$shareeIds,
			$shareTypes,
			$permissionsRole,
			$permissionsAction,
			$expirationDateTime
		);
		if ($response->getStatusCode() === 200) {
			$this->featureContext->shareNgAddToCreatedUserGroupShares($response);
		}
		return $response;
	}

	/**
	 * share the drive (space) using the drives.root endpoint
	 *
	 * @param string $user
	 * @param TableNode $table
	 * @param bool $federatedShare
	 *
	 * @return ResponseInterface
	 *
	 * @throws JsonException
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function sendDriveShareInvitation(
		string $user,
		TableNode $table,
		bool $federatedShare = false
	): ResponseInterface {
		$shareeIds = [];
		$rows = $table->getRowsHash();
		$space = $this->spacesContext->getSpaceByName($user, $rows['space']);
		$spaceId = $space['id'];

		$sharees = array_map('trim', explode(',', $rows['sharee']));
		$shareTypes = array_map('trim', explode(',', $rows['shareType']));

		foreach ($sharees as $index => $sharee) {
			$shareType = $shareTypes[$index];
			if ($sharee === "") {
				// set empty value to $shareeIds
				$shareeIds[] = "";
				continue;
			}
			$shareeId = "";
			if ($shareType === "user") {
				$shareeId = $this->featureContext->getAttributeOfCreatedUser($sharee, 'id');
				if ($federatedShare) {
					$federatedUser = $this->featureContext->ocmContext->getAcceptedUserByName(
						$user,
						$sharee
					);
					$shareeId = $federatedUser['user_id'] . "@" . $federatedUser['idp'];
				}
			} elseif ($shareType === "group") {
				$shareeId = $this->featureContext->getAttributeOfCreatedGroup($sharee, 'id');
			}
			// for non-existing group or user, generate random id
			$shareeIds[] = $shareeId ?: WebDavHelper::generateUUIDv4();
		}

		$permissionsRole = $rows['permissionsRole'] ?? null;
		$permissionsAction = $rows['permissionsAction'] ?? null;
		$expirationDateTime = (\array_key_exists('expirationDateTime', $rows))
			? \date(DATE_RFC3339, \strtotime($rows['expirationDateTime'])) : null;

		return GraphHelper::sendSharingInvitationForDrive(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$spaceId,
			$shareeIds,
			$shareTypes,
			$permissionsRole,
			$permissionsAction,
			$expirationDateTime
		);
	}

	/**
	 * @Given /^user "([^"]*)" has sent the following resource share invitation:$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function userHasSentTheFollowingResourceShareInvitation(string $user, TableNode $table): void {
		$rows = $table->getRowsHash();
		Assert::assertArrayHasKey(
			"resource",
			$rows,
			"'resource' should be provided in the data-table while sharing a resource"
		);
		$response = $this->sendShareInvitation($user, $rows);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
	}

	/**
	 * @Given /^user "([^"]*)" has sent the following resource share invitation to federated user:$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function userHasSentTheFollowingResourceShareInvitationToFederatedUser(
		string $user,
		TableNode $table
	): void {
		$rows = $table->getRowsHash();
		Assert::assertArrayHasKey(
			"resource",
			$rows,
			"'resource' should be provided in the data-table while sharing a resource"
		);
		$response = $this->sendShareInvitation($user, $rows, null, true);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
	}

	/**
	 * @Given /^user "([^"]*)" has sent the following space share invitation:$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function userHasSentTheFollowingShareShareInvitation(string $user, TableNode $table): void {
		$rows = $table->getRowsHash();
		Assert::assertArrayNotHasKey(
			"resource",
			$rows,
			"'resource' should not be provided in the data-table while sharing a space"
		);
		$response = $this->sendDriveShareInvitation($user, $table);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
	}

	/**
	 * @When /^user "([^"]*)" sends the following resource share invitation using the Graph API:$/
	 * @When /^user "([^"]*)" tries to send the following resource share invitation using the Graph API:$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function userSendsTheFollowingResourceShareInvitationUsingTheGraphApi(string $user, TableNode $table): void {
		$rows = $table->getRowsHash();
		Assert::assertArrayHasKey(
			"resource",
			$rows,
			"'resource' should be provided in the data-table while sharing a resource"
		);
		$this->featureContext->setResponse(
			$this->sendShareInvitation($user, $rows)
		);
	}

	/**
	 * @When /^user "([^"]*)" sends the following resource share invitation to federated user using the Graph API:$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function userSendsTheFollowingResourceShareInvitationToFederatedUserUsingTheGraphApi(
		string $user,
		TableNode $table
	): void {
		$rows = $table->getRowsHash();
		Assert::assertArrayHasKey(
			"resource",
			$rows,
			"'resource' should be provided in the data-table while sharing a resource"
		);
		$this->featureContext->setResponse(
			$this->sendShareInvitation($user, $rows, null, true)
		);
	}

	/**
	 * @When /^user "([^"]*)" sends the following space share invitation using permissions endpoint of the Graph API:$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function userSendsTheFollowingSpaceShareInvitationUsingPermissionsEndpointOfTheGraphApi(
		string $user,
		TableNode $table
	): void {
		$rows = $table->getRowsHash();
		Assert::assertArrayNotHasKey(
			"resource",
			$rows,
			"'resource' should not be provided in the data-table while sharing a space"
		);
		$this->featureContext->setResponse(
			$this->sendShareInvitation($user, $rows)
		);
	}

	/**
	 * @When /^user "([^"]*)" tries to send the following space share invitation to federated user using permissions endpoint of the Graph API:$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function userSendsTheFollowingSpaceShareInvitationToFederatedUserUsingPermissionsEndpointOfTheGraphApi(
		string $user,
		TableNode $table
	): void {
		$rows = $table->getRowsHash();
		Assert::assertArrayNotHasKey(
			"resource",
			$rows,
			"'resource' should not be provided in the data-table while sharing a space"
		);
		$this->featureContext->setResponse(
			$this->sendShareInvitation($user, $rows, null, true)
		);
	}

	/**
	 * @Given user :user has updated the last resource share with the following properties:
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 */
	public function userHasUpdatedTheLastResourceShareWithTheFollowingProperties(string $user, TableNode $table): void {
		$permissionID = $this->featureContext->shareNgGetLastCreatedUserGroupShareID();
		$response = $this->updateResourceShare(
			$user,
			$table,
			$permissionID
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "Expected response status code should be 200", $response);
	}

	/**
	 * @When user :user updates the last resource share with the following properties using the Graph API:
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 */
	public function userUpdatesTheLastShareWithFollowingPropertiesUsingGraphApi($user, TableNode $table) {
		$permissionID = $this->featureContext->shareNgGetLastCreatedUserGroupShareID();
		$this->featureContext->setResponse(
			$this->updateResourceShare(
				$user,
				$table,
				$permissionID
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" updates the space share for (user|group) "([^"]*)" with the following using the Graph API:$/
	 *
	 * @param string $user
	 * @param string $shareType
	 * @param string $sharee
	 * @param TableNode $table
	 *
	 * @return void
	 */
	public function userUpdatesTheSpaceShareForUserOrGroupWithFollowingUsingGraphApi(
		string $user,
		string $shareType,
		string $sharee,
		TableNode $table
	) {
		$permissionID = "";
		if ($shareType === "user") {
			$permissionID = "u:" . $this->featureContext->getAttributeOfCreatedUser($sharee, 'id');
		} elseif ($shareType === "group") {
			$permissionID = "g:" . $this->featureContext->getAttributeOfCreatedGroup($sharee, 'id');
		}

		$this->featureContext->setResponse(
			$this->updateResourceShare(
				$user,
				$table,
				$permissionID
			)
		);
	}

	/**
	 * @param string $user
	 * @param TableNode $body
	 * @param string $permissionID
	 *
	 * @return ResponseInterface
	 */
	public function updateResourceShare(string $user, TableNode  $body, string $permissionID): ResponseInterface {
		$bodyRows = $body->getRowsHash();
		$space = $this->spacesContext->getSpaceByName($user, $bodyRows['space']);
		$spaceId = $space["id"];
		// for updating role of project space shared, we do not need to provide resource
		$resource = $bodyRows['resource'] ?? '';
		if ($resource === '' && !\in_array($bodyRows['space'], ['Personal', 'Shares'])) {
			$itemId = $space['fileId'];
		} else {
			$itemId = $this->spacesContext->getResourceId($user, $bodyRows['space'], $resource);
		}
		$body = [];
		if (\array_key_exists('permissionsRole', $bodyRows)) {
			$body['roles'] = [GraphHelper::getPermissionsRoleIdByName($bodyRows['permissionsRole'])];
		}

		if (\array_key_exists('expirationDateTime', $bodyRows)) {
			$body['expirationDateTime'] = empty($bodyRows['expirationDateTime'])
			? null : $bodyRows['expirationDateTime'];
		}

		return GraphHelper::updateShare(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$spaceId,
			$itemId,
			\json_encode($body),
			$permissionID
		);
	}

	/**
	 * @When user :user sends the following share invitation with file-id :fileId using the Graph API:
	 *
	 * @param string $user
	 * @param string $fileId
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userSendsTheFollowingShareInvitationWithFileIdUsingTheGraphApi(
		string $user,
		string $fileId,
		TableNode $table
	): void {
		$rows = $table->getRowsHash();
		$this->featureContext->setResponse(
			$this->sendShareInvitation($user, $rows, $fileId)
		);
	}

	/**
	 * @When /^user "([^"]*)" creates the following resource link share using the Graph API:$/
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userCreatesAPublicLinkShareWithSettings(string $user, TableNode  $body): void {
		$response = $this->createLinkShare($user, $body);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When /^user "([^"]*)" (?:tries to create|creates) the following space link share using permissions endpoint of the Graph API:$/
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userCreatesTheFollowingSpaceLinkShareUsingPermissionsEndpointOfTheGraphApi(
		string $user,
		TableNode $body
	): void {
		$this->featureContext->setResponse($this->createLinkShare($user, $body));
	}

	/**
	 * @Given /^user "([^"]*)" has created the following resource link share:$/
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasCreatedTheFollowingResourceLinkShare(string $user, TableNode  $body): void {
		$rows = $body->getRowsHash();
		Assert::assertArrayHasKey(
			"resource",
			$rows,
			"'resource' should be provided in the data-table while sharing a resource"
		);
		$response = $this->createLinkShare($user, $body);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "Failed while creating public share link!", $response);
		$this->featureContext->shareNgAddToCreatedLinkShares($response);
	}

	/**
	 * @Given /^user "([^"]*)" has created the following space link share:$/
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasCreatedTheFollowingLinkShare(string $user, TableNode  $body): void {
		$rows = $body->getRowsHash();
		Assert::assertArrayNotHasKey(
			"resource",
			$rows,
			"'resource' should not be provided in the data-table while sharing a space"
		);
		$response = $this->createDriveLinkShare($user, $body);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "Failed while creating public share link!", $response);
		$this->featureContext->shareNgAddToCreatedLinkShares($response);
	}

	/**
	 * @Given user :user has updated the last resource/space link share with
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userHasUpdatedLastPublicLinkShare(string $user, TableNode  $body): void {
		$response = $this->updateLinkShare(
			$user,
			$body,
			$this->featureContext->shareNgGetLastCreatedLinkShareID()
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "Failed while updating public share link!", $response);
	}

	/**
	 * @When user :user updates the last public link share using the permissions endpoint of the Graph API:
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userUpdatesTheLastPublicLinkShareUsingThePermissionsEndpointOfTheGraphApi(
		string $user,
		TableNode  $body
	): void {
		$this->featureContext->setResponse(
			$this->updateLinkShare(
				$user,
				$body,
				$this->featureContext->shareNgGetLastCreatedLinkShareID()
			)
		);
	}

	/**
	 * @param string $user
	 * @param TableNode $body
	 * @param string $permissionID
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function updateLinkShare(string $user, TableNode  $body, string $permissionID): ResponseInterface {
		$bodyRows = $body->getRowsHash();
		$space = $bodyRows['space'];
		if (isset($bodyRows['resource'])) {
			$itemId = $this->spacesContext->getResourceId($user, $space, $bodyRows['resource']);
		} else {
			$itemId = $this->spacesContext->getResourceId($user, $space, $space);
		}
		$spaceId = ($this->spacesContext->getSpaceByName($user, $space))['id'];
		$body = [];

		if (\array_key_exists('permissionsRole', $bodyRows)) {
			$body['link']['type'] = $bodyRows['permissionsRole'];
		}

		if (\array_key_exists('expirationDateTime', $bodyRows)) {
			$body['expirationDateTime'] = empty($bodyRows['expirationDateTime'])
			? null : $bodyRows['expirationDateTime'];
		}

		if (\array_key_exists('displayName', $bodyRows)) {
			$body['displayName'] = $bodyRows['displayName'];
		}

		return GraphHelper::updateShare(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$spaceId,
			$itemId,
			\json_encode($body),
			$permissionID
		);
	}

	/**
	 * @param string $user
	 * @param TableNode $body
	 * @param string $permissionID
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function setLinkSharePassword(string $user, TableNode  $body, string $permissionID): ResponseInterface {
		$bodyRows = $body->getRowsHash();
		$space = $bodyRows['space'];
		$resource = $bodyRows['resource'];
		$spaceId = ($this->spacesContext->getSpaceByName($user, $space))["id"];
		$itemId = $this->spacesContext->getResourceId($user, $space, $resource);

		if (\array_key_exists('password', $bodyRows)) {
			$body = [
				"password" => $this->featureContext->getActualPassword($bodyRows['password']),
			];
		} else {
			throw new Error('Password is missing to set for share link!');
		}

		return GraphHelper::setLinkSharePassword(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$spaceId,
			$itemId,
			\json_encode($body),
			$permissionID
		);
	}

	/**
	 * @Given user :user has set the following password for the last link share:
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userHasSetTheFollowingPasswordForTheLastLinkShare(string $user, TableNode  $body): void {
		$response = $this->setLinkSharePassword(
			$user,
			$body,
			$this->featureContext->shareNgGetLastCreatedLinkShareID()
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Failed while setting public share link password!",
			$response
		);
	}

	/**
	 * @When user :user sets the following password for the last link share using the Graph API:
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userSetsOrUpdatesFollowingPasswordForLastLinkShareUsingTheGraphApi(
		string $user,
		TableNode $body
	): void {
		$this->featureContext->setResponse(
			$this->setLinkSharePassword(
				$user,
				$body,
				$this->featureContext->shareNgGetLastCreatedLinkShareID()
			)
		);
	}

	/**
	 * Remove user|group|link share of item (resource) or drive (space) using drives.permissions endpoint
	 *
	 * @param string $sharer
	 * @param string $shareType (user|group|link)
	 * @param string $space
	 * @param string|null $resource
	 * @param string|null $recipient
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function removeAccessToSpaceItem(
		string $sharer,
		string $shareType,
		string $space,
		?string $resource = null,
		?string $recipient = null
	): ResponseInterface {
		$spaceId = ($this->spacesContext->getSpaceByName($sharer, $space))["id"];
		$itemId = (isset($resource)) ? $this->spacesContext->getResourceId($sharer, $space, $resource)
		: $this->spacesContext->getResourceId($sharer, $space, $space);

		$permissionID = "";

		// if resource is not provided then it indicates a space share
		// and the space shares are not stored
		// so build the permission-id using the user or group id
		if ($resource === null) {
			if ($shareType === "user") {
				$permissionID = "u:" . $this->featureContext->getAttributeOfCreatedUser($recipient, 'id');
			} elseif ($shareType === "group") {
				$permissionID = "g:" . $this->featureContext->getAttributeOfCreatedGroup($recipient, 'id');
			}
		} else {
			$permissionID = ($shareType === 'link')
				? $this->featureContext->shareNgGetLastCreatedLinkShareID()
				: $this->featureContext->shareNgGetLastCreatedUserGroupShareID();
		}

		return
			GraphHelper::removeAccessToSpaceItem(
				$this->featureContext->getBaseUrl(),
				$this->featureContext->getStepLineRef(),
				$sharer,
				$this->featureContext->getPasswordForUser($sharer),
				$spaceId,
				$itemId,
				$permissionID
			);
	}

	/**
	 * Remove user|group|link from drive (space) using drives.root endpoint
	 *
	 * @param string $sharer
	 * @param string $shareType (user|group|link)
	 * @param string $space
	 * @param string|null $recipient
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function removeAccessToSpace(
		string $sharer,
		string $shareType,
		string $space,
		?string $recipient = null
	): ResponseInterface {
		$spaceId = ($this->spacesContext->getSpaceByName($sharer, $space))["id"];

		$permissionID = match ($shareType) {
			'link' => $this->featureContext->shareNgGetLastCreatedLinkShareID(),
			'user' => 'u:' . $this->featureContext->getAttributeOfCreatedUser($recipient, 'id'),
			'group' => 'g:' . $this->featureContext->getAttributeOfCreatedGroup($recipient, 'id'),
			default => throw new Exception("shareType '$shareType' does not match user|group|link "),
		};

		return
			GraphHelper::removeAccessToSpace(
				$this->featureContext->getBaseUrl(),
				$this->featureContext->getStepLineRef(),
				$sharer,
				$this->featureContext->getPasswordForUser($sharer),
				$spaceId,
				$permissionID
			);
	}

	/**
	 * @Given /^user "([^"]*)" has removed the access of (user|group) "([^"]*)" from (?:file|folder|resource) "([^"]*)" of space "([^"]*)"$/
	 *
	 * @param string $sharer
	 * @param string $recipientType (user|group)
	 * @param string $recipient can be both user or group
	 * @param string $resource
	 * @param string $space
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userHasRemovedAccessOfUserOrGroupFromResourceOfSpace(
		string $sharer,
		string $recipientType,
		string $recipient,
		string $resource,
		string $space
	): void {
		$response = $this->removeAccessToSpaceItem($sharer, $recipientType, $space, $resource);
		$this->featureContext->theHTTPStatusCodeShouldBe(204, "", $response);
	}

	/**
	 * @When /^user "([^"]*)" removes the access of (user|group) "([^"]*)" from (?:file|folder|resource) "([^"]*)" of space "([^"]*)" using the Graph API$/
	 *
	 * @param string $sharer
	 * @param string $recipientType (user|group)
	 * @param string $recipient can be both user or group
	 * @param string $resource
	 * @param string $space
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userRemovesAccessOfUserOrGroupFromResourceOfSpaceUsingGraphAPI(
		string $sharer,
		string $recipientType,
		string $recipient,
		string $resource,
		string $space
	): void {
		$this->featureContext->setResponse(
			$this->removeAccessToSpaceItem($sharer, $recipientType, $space, $resource, $recipient)
		);
	}

	/**
	 * @When /^user "([^"]*)" removes the access of (user|group) "([^"]*)" from space "([^"]*)" using permissions endpoint of the Graph API$/
	 *
	 * @param string $sharer
	 * @param string $recipientType (user|group)
	 * @param string $recipient can be both user or group
	 * @param string $space
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userRemovesAccessOfUserOrGroupFromSpaceUsingPermissionsEndpointOfGraphAPI(
		string $sharer,
		string $recipientType,
		string $recipient,
		string $space
	): void {
		$this->featureContext->setResponse(
			$this->removeAccessToSpaceItem($sharer, $recipientType, $space, null, $recipient)
		);
	}

	/**
	 * @When /^user "([^"]*)" has removed the last link share of (?:file|folder) "([^"]*)" from space "([^"]*)"$/
	 *
	 * @param string $sharer
	 * @param string $resource
	 * @param string $space
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userHasRemovedTheLastLinkShareOfFileOrFolderFromSpace(
		string $sharer,
		string $resource,
		string $space
	): void {
		$response = $this->removeAccessToSpaceItem($sharer, 'link', $space, $resource);
		$this->featureContext->theHTTPStatusCodeShouldBe(204, "", $response);
	}

	/**
	 * @When /^user "([^"]*)" removes the link of (?:file|folder) "([^"]*)" from space "([^"]*)" using the Graph API$/
	 *
	 * @param string $sharer
	 * @param string $resource
	 * @param string $space
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userRemovesSharePermissionOfAResourceInLinkShareUsingGraphAPI(
		string $sharer,
		string $resource,
		string $space
	): void {
		$this->featureContext->setResponse(
			$this->removeAccessToSpaceItem($sharer, 'link', $space, $resource)
		);
	}

	/**
	 * @When /^user "([^"]*)" removes the access of (user|group) "([^"]*)" from space "([^"]*)" using root endpoint of the Graph API$/
	 * @When /^user "([^"]*)" tries to remove the access of (user|group) "([^"]*)" from space "([^"]*)" using root endpoint of the Graph API$/
	 *
	 * @param string $sharer
	 * @param string $recipientType (user|group)
	 * @param string $recipient can be both user or group
	 * @param string $space
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userRemovesAccessOfUserOrGroupFromSpaceUsingGraphAPI(
		string $sharer,
		string $recipientType,
		string $recipient,
		string $space
	): void {
		$this->featureContext->setResponse(
			$this->removeAccessToSpace($sharer, $recipientType, $space, $recipient)
		);
	}

	/**
	 * @When /^user "([^"]*)" removes the link from space "([^"]*)" using root endpoint of the Graph API$/
	 *
	 * @param string $sharer
	 * @param string $space
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userRemovesLinkFromSpaceUsingRootEndpointOfGraphAPI(
		string $sharer,
		string $space
	): void {
		$this->featureContext->setResponse(
			$this->removeAccessToSpace($sharer, 'link', $space)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has removed the access of (user|group) "([^"]*)" from space "([^"]*)"$/
	 *
	 * @param string $sharer
	 * @param string $recipientType (user|group)
	 * @param string $recipient can be both user or group
	 * @param string $space
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userHasRemovedAccessOfUserOrGroupFromSpace(
		string $sharer,
		string $recipientType,
		string $recipient,
		string $space
	): void {
		$response = $this->removeAccessToSpace($sharer, $recipientType, $space, $recipient);
		$this->featureContext->theHTTPStatusCodeShouldBe(204, "", $response);
	}

	/**
	 * @param string $sharee
	 * @param string $shareID
	 * @param bool $hide
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function hideOrUnhideSharedResource(string $sharee, string $shareID, bool $hide = true): ResponseInterface {
		$shareSpaceId = GraphHelper::SHARES_SPACE_ID;
		$itemId = $shareSpaceId . '!' . $shareID;
		$body['@UI.Hidden'] = $hide;
		return GraphHelper::hideOrUnhideShare(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$this->featureContext->getActualUsername($sharee),
			$this->featureContext->getPasswordForUser($sharee),
			$itemId,
			$shareSpaceId,
			$body
		);
	}

	/**
	 * @Given user :user has disabled sync of last shared resource
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userHasDisabledSyncOfLastSharedResource(string $user): void {
		$shareItemId = $this->featureContext->shareNgGetLastCreatedUserGroupShareID();
		$shareSpaceId = GraphHelper::SHARES_SPACE_ID;
		$itemId = $shareSpaceId . '!' . $shareItemId;
		$response = GraphHelper::disableShareSync(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			$itemId,
			$shareSpaceId,
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			204,
			__METHOD__ . " could not disable sync of last share",
			$response
		);
	}

	/**
	 * @When user :user disables sync of share :share using the Graph API
	 * @When user :user tries to disable sync of share :share using the Graph API
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userDisablesSyncOfShareUsingTheGraphApi(string $user): void {
		$shareItemId = $this->featureContext->shareNgGetLastCreatedUserGroupShareID();
		$shareSpaceId = GraphHelper::SHARES_SPACE_ID;
		$itemId = $shareSpaceId . '!' . $shareItemId;
		$response = GraphHelper::disableShareSync(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			$itemId,
			$shareSpaceId,
		);
		$this->featureContext->setResponse($response);
		$this->featureContext->pushToLastStatusCodesArrays();
		// disable check for client.synchronize
		$this->featureContext->rememberUserAutoSyncSetting($user, false);
	}

	/**
	 * @When user :user hides the shared resource :sharedResource using the Graph API
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function userHidesTheSharedResourceUsingTheGraphApi(string $user): void {
		$shareItemId = $this->featureContext->shareNgGetLastCreatedUserGroupShareID();
		$response = $this->hideOrUnhideSharedResource($user, $shareItemId);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @Given  user :user has hidden the share :sharedResource
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function userHasHiddenTheShare(string $user): void {
		$shareItemId = $this->featureContext->shareNgGetLastCreatedUserGroupShareID();
		$response = $this->hideOrUnhideSharedResource($user, $shareItemId);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, '', $response);
	}

	/**
	 * @When user :user unhides the shared resource :sharedResource using the Graph API
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function userUnhidesTheSharedResourceUsingTheGraphApi(string $user): void {
		$shareItemId = $this->featureContext->shareNgGetLastCreatedUserGroupShareID();
		$response = $this->hideOrUnhideSharedResource($user, $shareItemId, false);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When user :user enables sync of share :share offered by :offeredBy from :space space using the Graph API
	 *
	 * @param string $user
	 * @param string $share
	 * @param string $offeredBy
	 * @param string $space
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userEnablesSyncOfShareUsingTheGraphApi(
		string $user,
		string $share,
		string $offeredBy,
		string $space
	): void {
		$share = ltrim($share, '/');
		$itemId = $this->spacesContext->getResourceId($offeredBy, $space, $share);
		$shareSpaceId = GraphHelper::SHARES_SPACE_ID;
		$response =  GraphHelper::enableShareSync(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			$itemId,
			$shareSpaceId
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * step definition for enabling sync for items for non-existing group|user|space sharer
	 *
	 * @When user :user tries to enable share sync of a resource :resource using the Graph API
	 * @When user :user enables share sync of a resource :resource using the Graph API
	 *
	 * @param string $user
	 * @param string $resource
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userTriesToEnableShareSyncOfResourceUsingTheGraphApi(string $user, string $resource): void {
		$shareSpaceId = GraphHelper::SHARES_SPACE_ID;
		$itemId = ($resource === 'nonexistent') ? WebDavHelper::generateUUIDv4() : $resource;

		$response =  GraphHelper::enableShareSync(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			$itemId,
			$shareSpaceId
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When user :user tries to disable share sync of a resource :resource using the Graph API
	 *
	 * @param string $user
	 * @param string $resource
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userTriesToDisableShareSyncOfResourceUsingTheGraphApi(string $user, string $resource): void {
		$shareSpaceId = GraphHelper::SHARES_SPACE_ID;
		$shareID = ($resource === 'nonexistent') ? WebDavHelper::generateUUIDv4() : $resource;
		$itemId = $shareSpaceId . '!' . $shareID;
		$response =  GraphHelper::disableShareSync(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			$itemId,
			$shareSpaceId
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @param string $user
	 * @param string $resource
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function isShareSynced(string $user, string $resource): bool {
		$resource = \trim($resource, '/');
		$response = GraphHelper::getSharesSharedWithMe(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user)
		);

		$shares = $this->featureContext->getJsonDecodedResponse($response)["value"];
		$syncStatus = false;
		foreach ($shares as $share) {
			if ($share["name"] === $resource) {
				$syncStatus = $share["@client.synchronize"];
				break;
			}
		}
		Assert::assertIsBool($syncStatus, "'@client.synchronize' must be a boolean value");
		return $syncStatus;
	}

	/**
	 * @param string $user
	 * @param string $resource
	 * @param string $status
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function waitAndCheckShareSyncStatus(string $user, string $resource, string $status): void {
		$expected = $status === "enabled";

		// NOTE: Sharing is async so it might take some time for the share to be available.
		$retried = 0;
		do {
			$shareSynced = $this->isShareSynced($user, $resource);

			if ($shareSynced === $expected) {
				return;
			}

			$tryAgain = !$shareSynced && $retried < HttpRequestHelper::numRetriesOnHttpTooEarly();
			if ($tryAgain) {
				$retried += 1;
				echo "[INFO] Wait for share sync status...";
				// wait 500ms and try again
				\usleep(500 * 1000);
			}
		} while ($tryAgain);
		Assert::fail("[Timeout] Sync for share '$resource' was expected to be '$status' but was not");
	}

	/**
	 * @Then /^user "([^"]*)" has a share "([^"]*)" synced$/
	 *
	 * @param string $user
	 * @param string $resource
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userHasShareSynced(string $user, string $resource): void {
		$this->waitAndCheckShareSyncStatus($user, $resource, "enabled");
	}

	/**
	 * @Then /^user "([^"]*)" should have sync (enabled|disabled) for share "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $status
	 * @param string $resource
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userShouldHaveSyncEnabledOrDisabledForShare(string $user, string $status, string $resource): void {
		$this->waitAndCheckShareSyncStatus($user, $resource, $status);
	}

	/**
	 * @Then user :user should be able to send the following resource share invitation with all allowed permission roles
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function userShouldBeAbleToSendShareTheFollowingInvitationWithAllAllowedPermissionRoles(
		string $user,
		TableNode $table
	): void {
		$listPermissionResponse = $this->featureContext->getJsonDecodedResponseBodyContent();
		if (!isset($listPermissionResponse->{'@libre.graph.permissions.roles.allowedValues'})) {
			Assert::fail(
				"The following response does not contain '@libre.graph.permissions.roles.allowedValues' property:\n"
				. $listPermissionResponse
			);
		}
		Assert::assertNotEmpty(
			$listPermissionResponse->{'@libre.graph.permissions.roles.allowedValues'},
			"'@libre.graph.permissions.roles.allowedValues' should not be empty"
		);
		$allowedPermissionRoles = $listPermissionResponse->{'@libre.graph.permissions.roles.allowedValues'};
		// this info is needed for log to see which roles allowed and which were not when tests fail
		$shareInvitationRequestResult = "From the given allowed role lists from the permissions:\n";
		$areAllSendInvitationSuccessFullForAllowedRoles = true;
		$rows = $table->getRowsHash();
		// when sending share invitation for a project space, the resource to be shared is project space itself. So resource can be put as empty
		$resource = $rows['resource'] ?? '';
		$shareType = $rows['shareType'];
		$space = $rows['space'];
		//this details is needed for result logging purpose to determine whether the resource shared is a resource or a project space
		$resourceDetail = ($resource) ? "resource '" . $resource : "space '" . $space;
		foreach ($allowedPermissionRoles as $role) {
			//we should be able to send share invitation for each of the role allowed for the files/folders which are listed in permissions (allowed)
			$roleAllowed = GraphHelper::getPermissionNameByPermissionRoleId($role->id);
			$responseSendInvitation = $this->sendShareInvitation(
				$user,
				array_merge($rows, ['permissionsRole' => $roleAllowed])
			);
			$jsonResponseSendInvitation = $this->featureContext->getJsonDecodedResponseBodyContent(
				$responseSendInvitation
			);
			$httpsStatusCode = $responseSendInvitation->getStatusCode();
			if ($httpsStatusCode === 200 && !empty($jsonResponseSendInvitation->value)) {
				// remove the share so that the same user can be share for the next allowed roles
				$removePermissionsResponse = $this->removeAccessToSpaceItem($user, $shareType, $space, $resource);
				Assert::assertEquals(204, $removePermissionsResponse->getStatusCode());
			} else {
				$areAllSendInvitationSuccessFullForAllowedRoles = false;
				$shareInvitationRequestResult .= "\tShare invitation for " . $resourceDetail
				. "' with role '" . $roleAllowed . "' failed and was not allowed.\n";
			}
		}
		Assert::assertTrue($areAllSendInvitationSuccessFullForAllowedRoles, $shareInvitationRequestResult);
	}

	/**
	 * @When /^user "([^"]*)" (?:tries to list|lists) the permissions of space "([^"]*)" using root endpoint of the Graph API$/
	 *
	 * @param string $user
	 * @param string $space
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 *
	 */
	public function userListsThePermissionsOfDriveUsingRootEndPointOFTheGraphApi(string $user, string $space): void {
		$this->featureContext->setResponse($this->getDrivePermissionsList($user, $space));
	}

	/**
	 * @When /^user "([^"]*)" (?:tries to send|sends) the following space share invitation using root endpoint of the Graph API:$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userSendsTheFollowingShareInvitationUsingRootEndPointTheGraphApi(
		string $user,
		TableNode $table
	): void {
		$response = $this->sendDriveShareInvitation($user, $table);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When /^user "([^"]*)" tries to send the following space share invitation to federated user using root endpoint of the Graph API:$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userSendsTheFollowingShareInvitationToFederatedUserUsingRootEndPointTheGraphApi(
		string $user,
		TableNode $table
	): void {
		$response = $this->sendDriveShareInvitation($user, $table, true);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When user :user updates the last drive share with the following using root endpoint of the Graph API:
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userUpdatesTheLastDriveShareWithTheFollowingUsingRootEndpointOfTheGraphApi(
		string $user,
		TableNode $table
	): void {
		$bodyRows = $table->getRowsHash();
		$permissionID = match ($bodyRows['shareType']) {
			'user' => 'u:' . $this->featureContext->getAttributeOfCreatedUser($bodyRows['sharee'], 'id'),
			'group' => 'g:' . $this->featureContext->getAttributeOfCreatedGroup($bodyRows['sharee'], 'id'),
			default => throw new Exception("shareType {$bodyRows['shareType']} does not match user|group "),
		};
		$space = $bodyRows['space'];
		$spaceId = ($this->spacesContext->getSpaceByName($user, $space))["id"];
		$body = [];

		if (\array_key_exists('permissionsRole', $bodyRows)) {
			$body['roles'] = [GraphHelper::getPermissionsRoleIdByName($bodyRows['permissionsRole'])];
		}

		if (\array_key_exists('expirationDateTime', $bodyRows)) {
			$body['expirationDateTime'] = empty($bodyRows['expirationDateTime'])
			? null : $bodyRows['expirationDateTime'];
		}

		$this->featureContext->setResponse(
			GraphHelper::updateDriveShare(
				$this->featureContext->getBaseUrl(),
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$spaceId,
				\json_encode($body),
				$permissionID
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" (?:tries to create|creates) the following space link share using root endpoint of the Graph API:$/
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userCreatesTheFollowingSpaceLinkShareUsingRootEndpointOfTheGraphApi(
		string $user,
		TableNode $body
	): void {
		$rows = $body->getRowsHash();
		Assert::assertArrayNotHasKey(
			"resource",
			$rows,
			"'resource' should not be provided in the data-table while sharing a space"
		);
		$response = $this->createDriveLinkShare($user, $body);

		$this->featureContext->setResponse($response);
	}

	/**
	 * @When user :user sets the following password for the last space link share using root endpoint of the Graph API:
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userSetsTheFollowingPasswordForTheLastSpaceLinkShareUsingRootEndpointOfTheGraphAPI(
		string $user,
		TableNode $body
	): void {
		$rows = $body->getRowsHash();
		Assert::assertArrayNotHasKey(
			"resource",
			$rows,
			"'resource' should not be provided in the data-table while setting password in space shared link"
		);

		Assert::assertArrayHasKey("password", $rows, "'password' is missing in the data-table");
		$body = [
			"password" => $this->featureContext->getActualPassword($rows['password']),
		];

		$space = $rows['space'];
		$spaceId = ($this->spacesContext->getSpaceByName($user, $space))["id"];

		$response = GraphHelper::setDriveLinkSharePassword(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$spaceId,
			\json_encode($body),
			$this->featureContext->shareNgGetLastCreatedLinkShareID()
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When user :user tries to remove the link from space :space owned by :owner using root endpoint of the Graph API
	 *
	 * @param string $user
	 * @param string $space
	 * @param string $spaceOwner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userTriesToRemoveShareLinkOfSpaceOwnedByUsingRootEndpointOfTheGraphApi(
		string $user,
		string $space,
		string $spaceOwner
	): void {
		$permissionID = $this->featureContext->shareNgGetLastCreatedLinkShareID();
		$spaceId = ($this->spacesContext->getSpaceByName($spaceOwner, $space))["id"];

		$response = GraphHelper::removeAccessToSpace(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$spaceId,
			$permissionID
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @Then user :user should not have any :shareType permissions on space :space
	 *
	 * @param string $user
	 * @param string $shareType
	 * @param string $space
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userShouldNotHaveAnyPermissionsOnSpace(string $user, string $shareType, string $space): void {
		$response = $this->getDrivePermissionsList($user, $space);
		$responseBody = $this->featureContext->getJsonDecodedResponse($response);
		foreach ($responseBody['value'] as $value) {
			switch ($shareType) {
				case $shareType === 'link':
					Assert::assertArrayNotHasKey(
						'link',
						$value,
						$space . ' space should not have any link permissions but found ' . print_r($value, true)
					);
					break;
				case $shareType === "share":
					Assert::assertArrayNotHasKey(
						'grantedToV2',
						$value,
						$space . ' space should not have any share permissions but found ' . print_r($value, true)
					);
					break;
				default:
					Assert::fail('Invalid share type has been specified');
			}
		}
	}

	/**
	 * @Then user :user should be able to send the following space share invitation with all allowed permission roles using root endpoint of the Graph API
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 * @codingStandardsIgnoreStart
	 */
	public function userShouldBeAbleToSendTheFollowingSpaceShareInvitationWithAllAllowedPermissionRolesUsingRootEndpointOFTheGraphApi(
		// @codingStandardsIgnoreEnd
		string $user,
		TableNode $table
	): void {
		$listPermissionResponse = $this->featureContext->getJsonDecodedResponseBodyContent();
		if (!isset($listPermissionResponse->{'@libre.graph.permissions.roles.allowedValues'})) {
			Assert::fail(
				"The following response does not contain '@libre.graph.permissions.roles.allowedValues' property:\n"
				. $listPermissionResponse
			);
		}
		Assert::assertNotEmpty(
			$listPermissionResponse->{'@libre.graph.permissions.roles.allowedValues'},
			"'@libre.graph.permissions.roles.allowedValues' should not be empty"
		);
		$allowedPermissionRoles = $listPermissionResponse->{'@libre.graph.permissions.roles.allowedValues'};
		// this info is needed for log to see which roles allowed and which were not when tests fail
		$shareInvitationRequestResult = "From the given allowed role lists from the permissions:\n";
		$areAllSendInvitationSuccessFullForAllowedRoles = true;
		$rows = $table->getRowsHash();

		$shareType = $rows['shareType'];
		$space = $rows['space'];
		$recipient = $rows['sharee'];

		foreach ($allowedPermissionRoles as $role) {
			// we should be able to send share invitation for each of the roles allowed which are listed in permissions (allowed)
			$roleAllowed = GraphHelper::getPermissionNameByPermissionRoleId($role->id);
			$responseSendInvitation = $this->sendDriveShareInvitation(
				$user,
				new TableNode(array_merge($table->getTable(), [['permissionsRole', $roleAllowed]]))
			);
			$jsonResponseSendInvitation = $this->featureContext->getJsonDecodedResponseBodyContent(
				$responseSendInvitation
			);
			$httpsStatusCode = $responseSendInvitation->getStatusCode();
			if ($httpsStatusCode === 200 && !empty($jsonResponseSendInvitation->value)) {
				// remove the share so that the same user can be share for the next allowed roles
				$removePermissionsResponse = $this->removeAccessToSpace($user, $shareType, $space, $recipient);
				Assert::assertEquals(204, $removePermissionsResponse->getStatusCode());
			} else {
				$areAllSendInvitationSuccessFullForAllowedRoles = false;
				$shareInvitationRequestResult .= "\tShare invitation for " . $space . "' with role '"
				. $roleAllowed . "' failed and was not allowed.\n";
			}
		}
		Assert::assertTrue($areAllSendInvitationSuccessFullForAllowedRoles, $shareInvitationRequestResult);
	}

	/**
	 * @When user :user tries to list the permissions of space :space owned by :spaceOwner using root endpoint of the Graph API
	 *
	 * @param string $user
	 * @param string $space
	 * @param string $spaceOwner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userTriesToListThePermissionsOfSpaceOwnedByUsingRootEndpointOfTheGraphApi(
		string $user,
		string $space,
		string $spaceOwner
	): void {
		$this->featureContext->setResponse($this->getDrivePermissionsList($user, $space, null, $spaceOwner));
	}

	/**
	 * @When user :user removes the last link share of space :space using permissions endpoint of the Graph API
	 *
	 * @param string $user
	 * @param string $space
	 *
	 * @return void
	 */
	public function userRemovesTheLastLinkShareOfSpaceUsingPermissionsEndpointOfGraphApi(
		string $user,
		string $space
	): void {
		$this->featureContext->setResponse(
			$this->removeAccessToSpaceItem($user, 'link', $space, '')
		);
	}

	/**
	 * @param string $share
	 * @param string $sharee
	 * @param string $sharer
	 * @param string $space
	 * @param bool $shouldExist
	 * @param bool $federatedShare
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 * @throws Exception
	 */
	public function checkIfShareExists(
		string $share,
		string $sharee,
		string $sharer,
		string $space,
		bool $shouldExist = true,
		bool $federatedShare = false
	): void {
		$share = \ltrim($share, "/");
		if (\strtolower($space) === "personal") {
			$remoteDriveAlias = "personal/" . \strtolower($sharer);
		} else {
			$remoteDriveAlias = "project/" . \strtolower($space);
		}

		if (!$federatedShare) {
			// check share mountpoint
			$response = GraphHelper::getMySpaces(
				$this->featureContext->getBaseUrl(),
				$sharee,
				$this->featureContext->getPasswordForUser($sharee),
				"",
				$this->featureContext->getStepLineRef()
			);
			$driveList = HttpRequestHelper::getJsonDecodedResponseBodyContent($response)->value;
			$foundShareMountpoint = false;
			foreach ($driveList as $drive) {
				if ($drive->driveType === "mountpoint"
					&& $drive->name === $share
					&& $drive->root->remoteItem->driveAlias === $remoteDriveAlias
				) {
					$foundShareMountpoint = true;
				}
			}
			Assert::assertSame(
				$shouldExist,
				$foundShareMountpoint,
				"Share mountpoint '$share' was not found in the drives list."
			);
		}

		// check share in shared-with-me list
		$response = GraphHelper::getSharesSharedWithMe(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$sharee,
			$this->featureContext->getPasswordForUser($sharee)
		);
		$sharedWithMeList = HttpRequestHelper::getJsonDecodedResponseBodyContent($response)->value;
		$foundShareInSharedWithMe = false;
		foreach ($sharedWithMeList as $item) {
			if ($item->name === $share) {
				foreach ($item->remoteItem->permissions as $permission) {
					$shareCreator = $permission->invitation->invitedBy->user->displayName;
					if ($shareCreator === $this->featureContext->getDisplayNameForUser($sharer)) {
						$foundShareInSharedWithMe = true;
						break;
					}
				}
				break;
			}
		}
		Assert::assertSame(
			$shouldExist,
			$foundShareInSharedWithMe,
			"Share '$share' was not found in the shared-with-me list"
		);
	}

	/**
	 * @Then /^user "([^"]*)" (should|should not) have a share "([^"]*)" shared by user "([^"]*)" from space "([^"]*)"$/
	 *
	 * @param string $sharee
	 * @param string $shouldOrNot
	 * @param string $share
	 * @param string $sharer
	 * @param string $space
	 *
	 * @return void
	 */
	public function userShouldHaveShareSharedByUserFromSpace(
		string $sharee,
		string $shouldOrNot,
		string $share,
		string $sharer,
		string $space
	): void {
		$this->checkIfShareExists($share, $sharee, $sharer, $space, $shouldOrNot === "should");
	}

	/**
	 * @Then /^user "([^"]*)" (should|should not) have a federated share "([^"]*)" shared by user "([^"]*)" from space "([^"]*)"$/
	 *
	 * @param string $sharee
	 * @param string $shouldOrNot
	 * @param string $share
	 * @param string $sharer
	 * @param string $space
	 *
	 * @return void
	 */
	public function userShouldOrShouldNotHaveFederatedShareSharedByUserFromSpace(
		string $sharee,
		string $shouldOrNot,
		string $share,
		string $sharer,
		string $space
	): void {
		$this->checkIfShareExists($share, $sharee, $sharer, $space, $shouldOrNot === "should", true);
	}

	/**
	 * @Given /^user "([^"]*)" has shared the following (?:files|folders) from space "([^"]*)" with user "([^"]*)" and role "([^"]*)":$/
	 *
	 * @param string $sharer
	 * @param string $space
	 * @param string $sharee
	 * @param string $role
	 * @param TableNode $table
	 *
	 * @return void
	 */
	public function userHasSharedTheFollowingFilesFromSpaceWithUserAndRole(
		string $sharer,
		string $space,
		string $sharee,
		string $role,
		TableNode $table
	): void {
		$rows = $table->getRows();
		foreach ($rows as $row) {
			if (isset($row[0])) {
				$shareData = [
					'resource' => $row[0],
					'space' => $space,
					'sharee' => $sharee,
					'shareType' => 'user',
					'permissionsRole' => $role,
				];
				$response = $this->sendShareInvitation($sharer, $shareData);
				$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
			}
		}
	}

	/**
	 * @Then the json response should contain the following shares:
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 */
	public function theJsonResponseShouldContainTheFollowingShares(TableNode $table): void {
		$responseBody = $this->featureContext->getJsonDecodedResponseBodyContent();

		$resourceNames = [];
		if (isset($responseBody->value) && \is_array($responseBody->value)) {
			foreach ($responseBody->value as $item) {
				if (isset($item->name)) {
					$resourceNames[] = $item->name;
				}
			}
		}

		$expectedShares = $table->getColumn(0);

		foreach ($expectedShares as $expectedShare) {
			Assert::assertTrue(
				\in_array($expectedShare, $resourceNames),
				"The share '$expectedShare' was not found in the response."
			);
		}
	}

	/**
	 * @When /^user "([^"]*)" gets the allowed roles for federated user of (folder|file) "([^"]*)" from the space "([^"]*)" using the Graph API$/
	 *
	 * @param string $user
	 * @param string $fileOrFolder (file|folder)
	 * @param string $resource
	 * @param string $space
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userGetsFederatedPermissionsListForFileOfTheSpaceUsingTheGraphApi(
		string $user,
		string $fileOrFolder,
		string $resource,
		string $space
	): void {
		$query = '$filter=@libre.graph.permissions.roles.allowedValues'
			. '/rolePermissions/any(p:contains(p/condition,+\'@Subject.UserType=="Federated"\'))'
			. '&$select=@libre.graph.permissions.roles.allowedValues';
		$this->featureContext->setResponse(
			$this->getPermissionsList($user, $fileOrFolder, $space, $resource, $query)
		);
	}

	/**
	 * @When /^user "([^"]*)" gets the permittion list of (folder|file) "([^"]*)" from the space "([^"]*)" using the Graph API with query "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $fileOrFolder (file|folder)
	 * @param string $resource
	 * @param string $space
	 * @param string $query
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userGetsPermissionsListWithQueryForFileOfTheSpaceUsingTheGraphApi(
		string $user,
		string $fileOrFolder,
		string $resource,
		string $space,
		string $query
	): void {
		$this->featureContext->setResponse(
			$this->getPermissionsList($user, $fileOrFolder, $space, $resource, $query)
		);
	}

	/**
	 * @When /^user "([^"]*)" gets the drive permittion list of the space "([^"]*)" using the Graph API with query "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $space
	 * @param string $query
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userGetsDrivePermissionsListWithQueryUsingTheGraphApi(
		string $user,
		string $space,
		string $query
	): void {
		$this->featureContext->setResponse($this->getDrivePermissionsList($user, $space, $query));
	}

	/**
	 * @Then /^the JSON data of the response should (not |)contain the following keys:$/
	 *
	 * @param string|null $shouldOrNot (not| )
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theJsonDataResponseShouldOrNotContainData(string $shouldOrNot, TableNode $table): void {
		$response = $this->featureContext->getJsonDecodedResponse($this->featureContext->getResponse());

		foreach ($table->getColumn(0) as $key) {
			$keyExists = \array_key_exists($key, $response);

			if (\trim($shouldOrNot) !== "not") {
				Assert::assertTrue(
					$keyExists,
					"Expected key '$key' to exist in the JSON response, but it doesn't."
				);
			} else {
				Assert::assertFalse(
					$keyExists,
					"Key '$key' should not exist in the JSON response, but it does."
				);
			}
		}
	}
}
