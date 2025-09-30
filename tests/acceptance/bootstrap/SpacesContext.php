<?php

declare(strict_types=1);
/**
 * @author Michael Barz <mbarz@owncloud.com>
 * @copyright Copyright (c) 2021 Michael Barz mbarz@owncloud.com
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License,
 * as published by the Free Software Foundation;
 * either version 3 of the License, or any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>
 *
 */

use Behat\Behat\Context\Context;
use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use Behat\Gherkin\Node\PyStringNode;
use Behat\Gherkin\Node\TableNode;
use GuzzleHttp\Exception\GuzzleException;
use PHPUnit\Framework\Assert;
use Psr\Http\Message\ResponseInterface;
use TestHelpers\HttpRequestHelper;
use TestHelpers\WebDavHelper;
use TestHelpers\GraphHelper;
use TestHelpers\OcHelper;
use TestHelpers\BehatHelper;

require_once 'bootstrap.php';

/**
 * Context for OpenCloud spaces specific steps
 */
class SpacesContext implements Context {
	private FeatureContext $featureContext;
	private OCSContext $ocsContext;
	private TrashbinContext $trashbinContext;
	private WebDavPropertiesContext $webDavPropertiesContext;
	private FavoritesContext $favoritesContext;
	private ChecksumContext $checksumContext;
	private FilesVersionsContext $filesVersionsContext;
	private ArchiverContext $archiverContext;

	/**
	 * key is space name and value is the username that created the space
	 */
	private array $createdSpaces = [];
	private string $ocsApiUrl = '/ocs/v2.php/apps/files_sharing/api/v1/shares';

	/**
	 * @var array map with user as key, spaces and file etags as value
	 * @example
	 * [
	 *   "user1" => [
	 *     "Personal" => [
	 *       "file1.txt": "etag1",
	 *     ],
	 *     "Shares" => []
	 *   ],
	 *   "user2" => [
	 *     ...
	 *   ]
	 * ]
	 */
	private array $storedEtags = [];

	/**
	 * @param string $spaceName
	 *
	 * @return string name of the user that created the space
	 * @throws Exception
	 */
	public function getSpaceCreator(string $spaceName): string {
		if (!\array_key_exists($spaceName, $this->createdSpaces)) {
			throw new Exception(__METHOD__ . " space '$spaceName' has not been created in this scenario");
		}
		return $this->createdSpaces[$spaceName]['spaceCreator'];
	}

	/**
	 * @param string $spaceCreator
	 * @param ResponseInterface $response
	 *
	 * @return void
	 */
	public function addCreatedSpace(string $spaceCreator, ResponseInterface $response): void {
		$response = $this->featureContext->getJsonDecodedResponseBodyContent($response);
		$spaceName = $response->name;
		$this->createdSpaces[$spaceName] = [];
		$this->createdSpaces[$spaceName]['name'] = $response->name;
		$this->createdSpaces[$spaceName]['id'] = $response->id;
		$this->createdSpaces[$spaceName]['spaceCreator'] = $spaceCreator;
		$this->createdSpaces[$spaceName]['fileId'] = $response->id . '!' . $response->owner->user->id;
	}

	/**
	 * @param string $spaceName
	 *
	 * @return array|null
	 */
	public function getCreatedSpace(string $spaceName): ?array {
		return $this->createdSpaces[$spaceName] ?? null;
	}

	private array $availableSpaces = [];

	/**
	 * @return array
	 */
	public function getAvailableSpaces(): array {
		return $this->availableSpaces;
	}

	/**
	 * @param array $availableSpaces
	 *
	 * @return void
	 */
	public function setAvailableSpaces(array $availableSpaces): void {
		$this->availableSpaces = $availableSpaces;
	}

	/**
	 * Get SpaceId by Name
	 *
	 * @param $name string
	 *
	 * @return string
	 *
	 * @throws Exception
	 */
	public function getSpaceIdByNameFromResponse(string $name): string {
		$space = $this->getSpaceByNameFromResponse($name);
		Assert::assertIsArray($space, "Space with name $name not found");
		if (!isset($space["id"])) {
			throw new Exception(__METHOD__ . " space with name $name not found");
		}
		return $space["id"];
	}

	/**
	 * Get Space Array by name
	 *
	 * @param string $name
	 * @param ResponseInterface|null $response
	 *
	 * @return array
	 *
	 * @throws Exception
	 */
	public function getSpaceByNameFromResponse(string $name, ?ResponseInterface $response = null): array {
		$response = $response ?? $this->featureContext->getResponse();
		$decodedResponse = $this->featureContext->getJsonDecodedResponse($response);
		$spaceAsArray = $decodedResponse;
		if (isset($decodedResponse['name']) && $decodedResponse['name'] === $name) {
			return $decodedResponse;
		}
		if (isset($spaceAsArray["value"])) {
			foreach ($spaceAsArray["value"] as $spaceCandidate) {
				if ($spaceCandidate['name'] === $name) {
					return $spaceCandidate;
				}
			}
		}
		return [];
	}

	/**
	 * The method finds available spaces to the user and returns the space by spaceName
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return array
	 * @throws GuzzleException
	 */
	public function getSpaceByName(string $user, string $spaceName): array {
		// First, try to find the space in locally saved (previously created) spaces — no API call
		$space = $this->getCreatedSpace($spaceName);
		if ($space !== null) {
			return $space;
		}
		// If it's a personal space and not found in the getCreatedSpace() — fetch via Graph API
		// GET /graph/v1.0/users/{userId}?$select=&$expand=drive
		if ($spaceName === "Personal" && $user !== 'admin') {
			return $this->getPersonalSpace($user);
		}
		// If the space is not found in the getCreatedSpace() and it's not personal — call API to list available spaces
		// Admin: GET /graph/v1.0/drives
		// Regular user: GET /graph/v1.0/me/drives
		if (strtolower($user) === 'admin') {
			$listSpacesFn = 'listAllAvailableSpaces';
		} else {
			$listSpacesFn = 'listAllAvailableSpacesOfUser';
		}

		// Sometimes listing available spaces might not return newly created/shared spaces.
		// So we try again until we find the space or we reach the max number of retries (i.e. 10)
		$retried = 0;
		do {
			// empty the available spaces array
			$this->setAvailableSpaces([]);

			$this->$listSpacesFn($user);
			$spaces = $this->getAvailableSpaces();

			$tryAgain = !\array_key_exists($spaceName, $spaces)
			&& $retried < HttpRequestHelper::numRetriesOnHttpTooEarly();
			if ($tryAgain) {
				$retried += 1;
				echo "Space '$spaceName' not found for user '$user', retrying ($retried)...\n";
				// wait 500ms and try again
				\usleep(500 * 1000);
			}
		} while ($tryAgain);

		Assert::assertArrayHasKey($spaceName, $spaces, "Space with name '$spaceName' for user '$user' not found");
		Assert::assertNotEmpty(
			$spaces[$spaceName]["root"]["webDavUrl"],
			"WebDavUrl for space with name '$spaceName' for user '$user' not found"
		);
		return $spaces[$spaceName];
	}

	/**
	 * @param string $user
	 *
	 * @return array
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function getPersonalSpace(string $user): array {
		$resource = GraphHelper::getUserWithDriveInformation(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$this->featureContext->getAttributeOfCreatedUser($user, 'id'),
			$this->featureContext->getStepLineRef()
		);

		$space = json_decode($resource->getBody()->getContents(), true);
		Assert::assertIsArray($space);
		Assert::assertArrayHasKey('drive', $space, "Drive information not found for user '$user'");
		Assert::assertArrayHasKey('id', $space['drive'], "Drive ID not found for user '$user'");

		return $space["drive"];
	}

	/**
	 * The method finds available spaces to the user and returns the spaceId by spaceName
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function getSpaceIdByName(string $user, string $spaceName): string {
		$space = $this->getSpaceByName($user, $spaceName);
		return $space["id"];
	}

	/**
	 * @param string $user
	 * @param string $share
	 *
	 * @return string
	 *
	 * @throws Exception|GuzzleException
	 */
	public function getSharesRemoteItemId(string $user, string $share): string {
		$credentials = $this->featureContext->graphContext->getAdminOrUserCredentials($user);
		$response = GraphHelper::getSharesSharedWithMe(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$credentials['username'],
			$credentials['password']
		);

		$jsonBody = $this->featureContext->getJsonDecodedResponseBodyContent($response);

		// Search remoteItem ID of a given share
		foreach ($jsonBody->value as $item) {
			if (isset($item->name) && $item->name === $share) {
				if (isset($item->remoteItem->id)) {
					return $item->remoteItem->id;
				}
				throw new Exception("Failed to find remoteItem ID for share: $share");
			}
		}

		throw new Exception("Cannot find share: $share");
	}

	/**
	 * The method finds file by fileName and spaceName and returns data of file which contains in responseHeader
	 * fileName contains the path, if the file is in the folder
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $fileName
	 * @param bool $federatedShare
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function getFileData(
		string $user,
		string $spaceName,
		string $fileName,
		bool $federatedShare = false
	): ResponseInterface {
		$baseUrl = $this->featureContext->getBaseUrl();

		if ($federatedShare) {
			$remoteItemId = $this->getSharesRemoteItemId($user, $spaceName);
			$spaceId = \rawurlencode($remoteItemId);
		} else {
			$space = $this->getSpaceByName($user, $spaceName);
			$spaceId = $space["id"];
		}

		$davPath = WebdavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $spaceId);
		$fullUrl = "$baseUrl/$davPath/$fileName";

		return HttpRequestHelper::get(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			[],
			"{}"
		);
	}

	/**
	 * The method returns fileId
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $fileName
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function getFileId(string $user, string $spaceName, string $fileName): string {
		$fileData = $this->getFileData($user, $spaceName, $fileName)->getHeaders();
		return $fileData["Oc-Fileid"][0];
	}

	/**
	 * The method returns "fileid" from the PROPFIND response
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $folderName
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function getResourceId(string $user, string $spaceName, string $folderName): string {
		$space = $this->getSpaceByName($user, $spaceName);
		// For a level 1 folder, the parent is space so $folderName = ''
		if ($folderName === $space["name"]) {
			$folderName = '';
		}

		$encodedName = \rawurlencode($folderName);
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebDavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $space["id"]);
		$fullUrl = "$baseUrl/$davPath/$encodedName";

		$response = HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			'PROPFIND',
			$user,
			$this->featureContext->getPasswordForUser($user),
			['Depth' => '0'],
		);

		$this->featureContext->theHttpStatusCodeShouldBe(207, '', $response);
		$responseXmlObject = HttpRequestHelper::getResponseXml($response, __METHOD__);
		$fileId = $responseXmlObject->xpath("//d:response/d:propstat/d:prop/oc:fileid")[0];
		return $fileId->__toString();
	}

	/**
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return string
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function getPrivateLink(string $user, string $spaceName): string {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$response = WebDavHelper::propfind(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			"",
			['oc:privatelink'],
			$this->featureContext->getStepLineRef(),
			"0",
			$spaceId,
			"files",
			WebDavHelper::DAV_VERSION_SPACES
		);
		$responseArray = json_decode(
			json_encode(
				HttpRequestHelper::getResponseXml($response, __METHOD__)
				->xpath("//d:response/d:propstat/d:prop/oc:privatelink")
			),
			true,
			512,
			JSON_THROW_ON_ERROR
		);
		Assert::assertNotEmpty($responseArray, "the PROPFIND response for $spaceName is empty");
		return $responseArray[0][0];
	}

	/**
	 * The method returns eTag
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $fileName
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function getETag(string $user, string $spaceName, string $fileName): string {
		$fileData = $this->getFileData($user, $spaceName, $fileName)->getHeaders();
		return \str_replace('"', '\"', $fileData["Etag"][0]);
	}

	/**
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return string
	 *
	 * @throws GuzzleException
	 */
	public function getEtagOfASpace(string $user, string $spaceName): string {
		return $this->getSpaceByName($user, $spaceName)["root"]["eTag"];
	}

	/**
	 * @BeforeScenario
	 *
	 * @param BeforeScenarioScope $scope
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function before(BeforeScenarioScope $scope): void {
		// Get the environment
		$environment = $scope->getEnvironment();
		// Get all the contexts you need in this context
		$this->featureContext = BehatHelper::getContext($scope, $environment, 'FeatureContext');
		$this->ocsContext = BehatHelper::getContext($scope, $environment, 'OCSContext');
		$this->trashbinContext = BehatHelper::getContext($scope, $environment, 'TrashbinContext');
		$this->webDavPropertiesContext = BehatHelper::getContext($scope, $environment, 'WebDavPropertiesContext');
		$this->favoritesContext = BehatHelper::getContext($scope, $environment, 'FavoritesContext');
		$this->checksumContext = BehatHelper::getContext($scope, $environment, 'ChecksumContext');
		$this->filesVersionsContext = BehatHelper::getContext($scope, $environment, 'FilesVersionsContext');
		$this->archiverContext = BehatHelper::getContext($scope, $environment, 'ArchiverContext');
	}

	/**
	 * @AfterScenario
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function cleanDataAfterTests(): void {
		if (OcHelper::isTestingOnReva() || OcHelper::isUsingPreparedLdapUsers()) {
			return;
		}
		$this->deleteAllProjectSpaces();
	}

	/**
	 * the admin user first disables and then deletes spaces
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function deleteAllProjectSpaces(): void {
		$query = "\$filter=driveType eq project";
		$userAdmin = $this->featureContext->getAdminUsername();

		$this->listAllAvailableSpaces(
			$userAdmin,
			$query
		);
		$drives = $this->getAvailableSpaces();

		foreach ($drives as $value) {
			if (!\array_key_exists("deleted", $value["root"])) {
				GraphHelper::disableSpace(
					$this->featureContext->getBaseUrl(),
					$userAdmin,
					$this->featureContext->getPasswordForUser($userAdmin),
					$value["id"],
					$this->featureContext->getStepLineRef()
				);
			}
			GraphHelper::deleteSpace(
				$this->featureContext->getBaseUrl(),
				$userAdmin,
				$this->featureContext->getPasswordForUser($userAdmin),
				$value["id"],
				$this->featureContext->getStepLineRef()
			);
		}
	}

	/**
	 * Send POST Request to url
	 *
	 * @param string $fullUrl
	 * @param string $user
	 * @param string $password
	 * @param mixed $body
	 * @param string $xRequestId
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public function sendPostRequestToUrl(
		string $fullUrl,
		string $user,
		string $password,
		$body,
		string $xRequestId = '',
		array $headers = []
	): ResponseInterface {
		return HttpRequestHelper::post($fullUrl, $xRequestId, $user, $password, $headers, $body);
	}

	/**
	 * Send Graph Create Folder Request
	 *
	 * @param  string $fullUrl
	 * @param  string $method
	 * @param  string $user
	 * @param  string $password
	 * @param  string $xRequestId
	 * @param  array  $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public function sendCreateFolderRequest(
		string $fullUrl,
		string $method,
		string $user,
		string $password,
		string $xRequestId = '',
		array $headers = []
	): ResponseInterface {
		return HttpRequestHelper::sendRequest($fullUrl, $xRequestId, $method, $user, $password, $headers);
	}

	/**
	 * @param string $user
	 * @param string $query
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function listAllAvailableSpacesOfUser(
		string $user,
		string $query = '',
		array $headers = []
	): ResponseInterface {
		$response = GraphHelper::getMySpaces(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			"?" . $query,
			$this->featureContext->getStepLineRef(),
			[],
			$headers
		);
		$this->rememberTheAvailableSpaces($response);
		return $response;
	}

	/**
	 * @When /^user "([^"]*)" lists all available spaces via the Graph API$/
	 * @When /^user "([^"]*)" lists all available spaces via the Graph API with query "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $query
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserListsAllHisAvailableSpacesUsingTheGraphApi(string $user, string $query = ''): void {
		$this->featureContext->setResponse($this->listAllAvailableSpacesOfUser($user, $query));
	}

	/**
	 * @When /^user "([^"]*)" lists all available spaces with headers using the Graph API$/
	 *
	 * @param string $user
	 * @param TableNode $headersTable
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserListsAllHisAvailableSpacesWithHeadersUsingTheGraphApi(
		string $user,
		TableNode $headersTable
	): void {
		$this->featureContext->verifyTableNodeColumns(
			$headersTable,
			['header', 'value']
		);
		$headers = [];
		foreach ($headersTable as $row) {
			$headers[$row['header']] = $row ['value'];
		}
		$this->featureContext->setResponse($this->listAllAvailableSpacesOfUser($user, '', $headers));
	}

	/**
	 * The method is used on the administration setting tab, which only the Admin user and the Space admin user have access to
	 *
	 * @param string $user
	 * @param string $query
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function listAllAvailableSpaces(string $user, string $query = ''): ResponseInterface {
		$response = GraphHelper::getAllSpaces(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			"?" . $query,
			$this->featureContext->getStepLineRef()
		);
		$this->rememberTheAvailableSpaces($response);
		return $response;
	}

	/**
	 * @When /^user "([^"]*)" lists all spaces via the Graph API$/
	 * @When /^user "([^"]*)" lists all spaces via the Graph API with query "([^"]*)"$/
	 * @When /^user "([^"]*)" tries to list all spaces via the Graph API$/
	 *
	 * @param string $user
	 * @param string $query
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserListsAllAvailableSpacesUsingTheGraphApi(string $user, string $query = ''): void {
		$this->featureContext->setResponse(
			$this->listAllAvailableSpaces($user, $query)
		);
	}

	/**
	 * @When /^user "([^"]*)" looks up the single space "([^"]*)" via the Graph API by using its id$/
	 * @When /^user "([^"]*)" tries to look up the single space "([^"]*)" owned by the user "([^"]*)" by using its id$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $ownerUser
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function theUserLooksUpTheSingleSpaceUsingTheGraphApiByUsingItsId(
		string $user,
		string $spaceName,
		string $ownerUser = ''
	): void {
		$space = $this->getSpaceByName(($ownerUser !== "") ? $ownerUser : $user, $spaceName);
		Assert::assertIsArray($space);
		Assert::assertNotEmpty($space["id"]);
		$this->featureContext->setResponse(
			GraphHelper::getSingleSpace(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$space["id"],
				'',
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" (?:creates|tries to create) a space "([^"]*)" of type "([^"]*)" with quota "([^"]*)" using the Graph API$/
	 * @When /^user "([^"]*)" (?:creates|tries to create) a space "([^"]*)" of type "([^"]*)" with the default quota using the Graph API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $spaceType
	 * @param int|null $quota
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserCreatesASpaceWithQuotaUsingTheGraphApi(
		string $user,
		string $spaceName,
		string $spaceType,
		?int $quota = null
	): void {
		$space = ["name" => $spaceName, "driveType" => $spaceType, "quota" => ["total" => $quota]];
		$body = json_encode($space);
		$this->featureContext->setResponse(
			$this->createSpace($user, $space, $body)
		);
	}

	/**
	 * Remember the available Spaces
	 *
	 * @param ResponseInterface|null $response
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function rememberTheAvailableSpaces(?ResponseInterface $response = null): void {
		if ($response) {
			$rawBody =  $response->getBody()->getContents();
		} else {
			$rawBody =  $this->featureContext->getResponse()->getBody()->getContents();
		}
		$drives = json_decode($rawBody, true, 512, JSON_THROW_ON_ERROR);
		if (isset($drives["value"])) {
			$drives = $drives["value"];
		}

		$spaces = [];
		foreach ($drives as $drive) {
			$spaces[$drive["name"]] = $drive;
		}
		$this->setAvailableSpaces($spaces);
	}

	/**
	 * @param string $user
	 * @param string $spaceName
	 * @param string $foldersPath
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function propfindSpace(string $user, string $spaceName, string $foldersPath = ''): ResponseInterface {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		return WebDavHelper::propfind(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$foldersPath,
			[],
			$this->featureContext->getStepLineRef(),
			null,
			$spaceId,
			'files',
			WebDavHelper::DAV_VERSION_SPACES
		);
	}

	/**
	 * @When /^user "([^"]*)" lists the content of the space with the name "([^"]*)" using the WebDav Api$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $foldersPath
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function theUserListsTheContentOfAPersonalSpaceRootUsingTheWebDAvApi(
		string $user,
		string $spaceName,
		string $foldersPath = ''
	): void {
		$this->featureContext->setResponse($this->propfindSpace($user, $spaceName, $foldersPath));
	}

	/**
	 * @When /^user "([^"]*)" sends PATCH request to the space "([^"]*)" of user "([^"]*)" with data "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $owner
	 * @param string $data
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userSendsPatchRequestToTheSpaceOfUserWithData(
		string $user,
		string $spaceName,
		string $owner,
		string $data
	): void {
		$space = $this->getSpaceByName($owner, $spaceName);
		Assert::assertIsArray($space);
		Assert::assertNotEmpty($spaceId = $space["id"]);
		$url = GraphHelper::getFullUrl($this->featureContext->getBaseUrl(), 'drives/' . $spaceId);
		$this->featureContext->setResponse(
			HttpRequestHelper::sendRequest(
				$url,
				$this->featureContext->getStepLineRef(),
				"PATCH",
				$this->featureContext->getActualUsername($user),
				$this->featureContext->getPasswordForUser($user),
				null,
				$data
			)
		);
	}

	/**
	 * @Then /^the (?:propfind|search) result of the space should (not|)\s?contain these (?:files|entries):$/
	 *
	 * @param string    $shouldOrNot   (not|)
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function thePropfindResultShouldContainEntries(
		string $shouldOrNot,
		TableNode $expectedFiles
	): void {
		$this->featureContext->propfindResultShouldContainEntries(
			$shouldOrNot,
			$expectedFiles,
		);
	}

	/**
	 * @Then /^for user "([^"]*)" the space "([^"]*)" should (not|)\s?contain these (?:files|entries):$/
	 *
	 * @param string    $user
	 * @param string    $spaceName
	 * @param string    $shouldOrNot   (not|)
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function userTheSpaceShouldContainEntries(
		string $user,
		string $spaceName,
		string $shouldOrNot,
		TableNode $expectedFiles
	): void {
		$space = $this->getSpaceByName($user, $spaceName);
		$this->featureContext->setResponse($this->propfindSpace($user, $spaceName));
		$this->featureContext->propfindResultShouldContainEntries(
			$shouldOrNot,
			$expectedFiles,
			$user,
			'PROPFIND',
			'',
			$space['id']
		);
	}

	/**
	 * @Then /^for user "([^"]*)" folder "([^"]*)" of the space "([^"]*)" should (not|)\s?contain these (?:files|entries):$/
	 *
	 * @param string    $user
	 * @param string    $folderPath
	 * @param string    $spaceName
	 * @param string    $shouldOrNot   (not|)
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function folderOfTheSpaceShouldContainEntries(
		string $user,
		string $folderPath,
		string $spaceName,
		string $shouldOrNot,
		TableNode $expectedFiles
	): void {
		$space = $this->getSpaceByName($user, $spaceName);
		$this->featureContext->setResponse($this->propfindSpace($user, $spaceName, $folderPath));
		$this->featureContext->propfindResultShouldContainEntries(
			$shouldOrNot,
			$expectedFiles,
			$this->featureContext->getActualUsername($user),
			'PROPFIND',
			$folderPath,
			$space['id']
		);
	}

	/**
	 * @Then /^for user "([^"]*)" the content of the file "([^"]*)" of the space "([^"]*)" should be "([^"]*)"$/
	 *
	 * @param string    $user
	 * @param string    $file
	 * @param string    $spaceName
	 * @param string    $fileContent
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function checkFileContent(
		string $user,
		string $file,
		string $spaceName,
		string $fileContent
	): void {
		$actualFileContent = $this->getFileData($user, $spaceName, $file)->getBody()->getContents();
		Assert::assertEquals($fileContent, $actualFileContent, "$file does not contain $fileContent");
	}

	/**
	 * @Then /^for user "([^"]*)" the content of file "([^"]*)" of federated share "([^"]*)" should be "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $share
	 * @param string $fileContent
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function forUserTheContentOfFileOfFederatedShareShouldBe(
		string $user,
		string $file,
		string $share,
		string $fileContent
	): void {
		$actualFileContent = $this->getFileData(
			$user,
			$share,
			$file,
			true
		)->getBody()->getContents();
		Assert::assertEquals($fileContent, $actualFileContent, "File content did not match");
	}

	/**
	 * @Then /^the JSON response should contain space called "([^"]*)" (?:|(?:owned by|granted to) "([^"]*)" )(?:|(?:with description file|with space image) "([^"]*)" )and match$/
	 *
	 * @param string $spaceName
	 * @param string|null $userName
	 * @param string|null $fileName
	 * @param PyStringNode|null $schemaString
	 *
	 * @return void
	 */
	public function theJsonDataFromLastResponseShouldMatch(
		string $spaceName,
		?string $userName = null,
		?string $fileName = null,
		?PyStringNode $schemaString = null
	): void {
		Assert::assertNotNull($schemaString, 'schema is not valid JSON');

		if (isset($this->featureContext->getJsonDecodedResponseBodyContent()->value)) {
			$responseBody = $this->featureContext->getJsonDecodedResponseBodyContent()->value;
			foreach ($responseBody as $value) {
				if (isset($value->name) && $value->name === $spaceName) {
					$responseBody = $value;
					break;
				}
			}
		} else {
			$responseBody = $this->featureContext->getJsonDecodedResponseBodyContent();
		}

		// substitute the value here
		if (\gettype($schemaString) !== 'string') {
			$schemaString = $schemaString->getRaw();
		}
		$schemaString = $this->featureContext->substituteInLineCodes(
			$schemaString,
			$this->featureContext->getCurrentUser(),
			[],
			[
				[
					"code" => "%space_id%",
					"function" =>
						[$this, "getSpaceIdByNameFromResponse"],
					"parameter" => [$spaceName]
				],
				[
					"code" => "%file_id%",
					"function" =>
						[$this, "getFileId"],
					"parameter" => [$userName, $spaceName, $fileName]
				],
				[
					"code" => "%eTag%",
					"function" =>
						[$this, "getETag"],
					"parameter" => [$userName, $spaceName, $fileName]
				],
				[
					"code" => "%space_etag%",
					"function" =>
						[$this, "getEtagOfASpace"],
					"parameter" => [$userName, $spaceName]
				]
			],
			null,
			$userName,
		);
		$this->featureContext->assertJsonDocumentMatchesSchema(
			$responseBody,
			$this->featureContext->getJSONSchema($schemaString)
		);
	}

	/**
	 * @Then /^the user "([^"]*)" should have a space called "([^"]*)" granted to "([^"]*)" with role "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $grantedUser
	 * @param string $role
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function checkPermissionsInResponse(
		string $user,
		string $spaceName,
		string $grantedUser,
		string $role
	): void {
		$response = $this->listAllAvailableSpacesOfUser($user, '$expand=root($expand=permissions)');
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200",
			$response
		);
		Assert::assertIsArray(
			$spaceAsArray = $this->getSpaceByNameFromResponse($spaceName, $response),
			"No space with name $spaceName found"
		);
		$permissions = $spaceAsArray["root"]["permissions"];
		$userId = $this->featureContext->getAttributeOfCreatedUser($grantedUser, 'id');

		$userRole = "";
		foreach ($permissions as $permission) {
			foreach ($permission["grantedToIdentities"] as $grantedToIdentities) {
				if ($grantedToIdentities["user"]["id"] === $userId) {
					$userRole = $permission["roles"][0];
				}
			}
		}
		Assert::assertEquals($userRole, $role, "the user $grantedUser with the role $role could not be found");
	}

	/**
	 * @Then /^the json response should not contain a space with name "([^"]*)"$/
	 *
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception
	 */
	public function jsonRespondedShouldNotContain(
		string $spaceName
	): void {
		Assert::assertEmpty(
			$this->getSpaceByNameFromResponse($spaceName),
			"space $spaceName should not be available for a user"
		);
	}

	/**
	 * @Then /^the user "([^"]*)" should (not |)have a space called "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $shouldOrNot
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldNotHaveSpace(
		string $user,
		string $shouldOrNot,
		string $spaceName
	): void {
		$response = $this->listAllAvailableSpacesOfUser($user);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200",
			$response
		);
		if (\trim($shouldOrNot) === "not") {
			Assert::assertEmpty(
				$this->getSpaceByNameFromResponse($spaceName, $response),
				"space $spaceName should not be available for a user"
			);
		} else {
			Assert::assertNotEmpty(
				$this->getSpaceByNameFromResponse($spaceName, $response),
				"space '$spaceName' should be available for a user '$user' but not found"
			);
		}
	}

	/**
	 * @Then /^the user "([^"]*)" should have a space "([^"]*)" in the disable state$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return mixed
	 * @throws Exception
	 */
	public function theUserShouldHaveASpaceInTheDisableState(
		string $user,
		string $spaceName
	): void {
		$response = $this->listAllAvailableSpacesOfUser($user);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200",
			$response
		);
		$decodedResponse = $this->featureContext->getJsonDecodedResponse($response);
		if (isset($decodedResponse["value"])) {
			foreach ($decodedResponse["value"] as $spaceCandidate) {
				if ($spaceCandidate['name'] === $spaceName) {
					if ($spaceCandidate['root']['deleted']['state'] !== 'trashed') {
						throw new \Exception(
							"space $spaceName should be in disable state but it's not "
						);
					}
					return;
				}
			}
		}
		throw new \Exception("space '$spaceName' should be available for a user '$user' but not found");
	}

	/**
	 * @Then /^the json response should (not|only|)\s?contain spaces of type "([^"]*)"$/
	 *
	 * @param string $onlyOrNot (not|only|)
	 * @param string $type
	 *
	 * @return void
	 * @throws Exception
	 */
	public function jsonRespondedShouldNotContainSpaceType(
		string $onlyOrNot,
		string $type
	): void {
		Assert::assertNotEmpty(
			$spaces = json_decode(
				(string)$this->featureContext
					->getResponse()->getBody(),
				true,
				512,
				JSON_THROW_ON_ERROR
			)
		);
		$matches = [];
		foreach ($spaces["value"] as $space) {
			if ($onlyOrNot === "not") {
				Assert::assertNotEquals($space["driveType"], $type);
			}
			if ($onlyOrNot === "only") {
				Assert::assertEquals($space["driveType"], $type);
			}
			if ($onlyOrNot === "" && $space["driveType"] === $type) {
				$matches[] = $space;
			}
		}
		if ($onlyOrNot === "") {
			Assert::assertNotEmpty($matches);
		}
	}

	/**
	 * Verify that the tableNode contains expected number of columns
	 *
	 * @param TableNode $table
	 * @param int $count
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function verifyTableNodeColumnsCount(
		TableNode $table,
		int $count
	): void {
		if (\count($table->getRows()) < 1) {
			throw new Exception("Table should have at least one row.");
		}
		$rowCount = \count($table->getRows()[0]);
		if ($count !== $rowCount) {
			throw new Exception("Table expected to have $count rows but found $rowCount");
		}
	}

	/**
	 * Escapes the given string for
	 * 1. Space --> %20
	 * 2. Opening Small Bracket --> %28
	 * 3. Closing Small Bracket --> %29
	 *
	 * @param string $path - File path to parse
	 *
	 * @return string
	 */
	public function escapePath(string $path): string {
		return \str_replace([" ", "(", ")"], ["%20", "%28", "%29"], $path);
	}

	/**
	 * @When /^user "([^"]*)" creates a (?:folder|subfolder) "([^"]*)" in space "([^"]*)" using the WebDav Api$/
	 *
	 * @param string $user
	 * @param string $folder
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function theUserCreatesAFolderUsingTheGraphApi(
		string $user,
		string $folder,
		string $spaceName
	): void {
		$folder = \trim($folder, '/');
		$exploded = explode('/', $folder);
		$path = '';
		for ($i = 0; $i < \count($exploded); $i++) {
			$path = $path . $exploded[$i] . '/';
			$response = $this->createFolderInSpace($user, $path, $spaceName);
			$this->featureContext->setResponse($response);
		}
	}

	/**
	 * @When /^user "([^"]*)" tries to create subfolder "([^"]*)" in a nonexistent folder of the space "([^"]*)" using the WebDav Api$/
	 *
	 * @param string $user
	 * @param string $subfolder
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function theUserTriesToCreateASubFolderUsingTheGraphApi(
		string $user,
		string $subfolder,
		string $spaceName
	): void {
		$response = $this->createFolderInSpace($user, $subfolder, $spaceName);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @Given /^user "([^"]*)" has created a (?:folder|subfolder) "([^"]*)" in space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $folder
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function userHasCreatedAFolderInSpace(
		string $user,
		string $folder,
		string $spaceName
	): void {
		$folder = \trim($folder, '/');
		$paths = explode('/', $folder);
		$folderPath = '';
		foreach ($paths as $path) {
			$folderPath .= "$path/";
			$response = $this->createFolderInSpace($user, $folderPath, $spaceName);
		}
		$this->featureContext->theHTTPStatusCodeShouldBe(
			201,
			"Expected response status code should be 201",
			$response
		);
	}

	/**
	 * @param string $user
	 * @param string $folder
	 * @param string $spaceName
	 * @param string $ownerUser
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public function createFolderInSpace(
		string $user,
		string $folder,
		string $spaceName,
		string $ownerUser = ''
	): ResponseInterface {
		if ($ownerUser === '') {
			$ownerUser = $user;
		}
		$spaceId = $this->getSpaceIdByName($ownerUser, $spaceName);
		return $this->featureContext->createFolder($user, $folder, false, null, $spaceId);
	}

	/**
	 * @When /^user "([^"]*)" creates a folder "([^"]*)" in space "([^"]*)" owned by the user "([^"]*)" using the WebDav Api$/
	 *
	 * @param string $user
	 * @param string $folder
	 * @param string $spaceName
	 * @param string $ownerUser
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function theUserCreatesAFolderToAnotherOwnerSpaceUsingTheGraphApi(
		string $user,
		string $folder,
		string $spaceName,
		string $ownerUser = ''
	): void {
		$response = $this->createFolderInSpace($user, $folder, $spaceName, $ownerUser);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When /^user "([^"]*)" uploads a file inside space "([^"]*)" with content "([^"]*)" to "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $content
	 * @param string $destination
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserUploadsAFileToSpace(
		string $user,
		string $spaceName,
		string $content,
		string $destination
	): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$response = $this->featureContext->uploadFileWithContent($user, $content, $destination, $spaceId);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When user :user updates the content of federated share :share with :content using the WebDAV API
	 *
	 * @param string $user
	 * @param string $share
	 * @param string $content
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function userUpdatesTheContentOfFederatedShareWithUsingTheWebdavApi(
		string $user,
		string $share,
		string $content,
	): void {
		$spaceId = $this->getSharesRemoteItemId($user, $share);
		$this->featureContext->setResponse(
			$this->featureContext->uploadFileWithContent(
				$user,
				$content,
				'',
				$spaceId
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" uploads a file "([^"]*)" to "([^"]*)" in space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserUploadsALocalFileToSpace(
		string $user,
		string $source,
		string $destination,
		string $spaceName
	): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$response = $this->featureContext->uploadFile($user, $source, $destination, $spaceId);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @Given user :user has uploaded a file :source to :destination in space :spaceName
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function userHasUploadedAFileToInSpaceUsingTheWebdavApi(
		string $user,
		string $source,
		string $destination,
		string $spaceName
	): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$response = $this->featureContext->uploadFile($user, $source, $destination, $spaceId);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			201,
			"Expected response status code should be 201",
			$response
		);
	}

	/**
	 * @When /^user "([^"]*)" uploads a file inside space "([^"]*)" owned by the user "([^"]*)" with content "([^"]*)" to "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $ownerUser
	 * @param string $content
	 * @param string $destination
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserUploadsAFileToAnotherOwnerSpace(
		string $user,
		string $spaceName,
		string $ownerUser,
		string $content,
		string $destination
	): void {
		$spaceId = $this->getSpaceIdByName($ownerUser, $spaceName);
		$response = $this->featureContext->uploadFileWithContent(
			$user,
			$content,
			$destination,
			$spaceId
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @param string $user
	 * @param string $spaceName
	 * @param array $bodyData
	 * @param string $owner
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function updateSpace(
		string $user,
		string $spaceName,
		array $bodyData,
		string $owner = ''
	): ResponseInterface {
		if ($spaceName === "non-existing") {
			// check sending invalid data
			$spaceId = "39c49dd3-1f24-4687-97d1-42df43f71713";
		} else {
			$space = $this->getSpaceByName(($owner !== "") ? $owner : $user, $spaceName);
			$spaceId = $space["id"];
		}

		$body = json_encode($bodyData, JSON_THROW_ON_ERROR);

		return GraphHelper::updateSpace(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$body,
			$spaceId,
			$this->featureContext->getStepLineRef()
		);
	}

	/**
	 * @param string $user
	 * @param string $targetUser
	 * @param array $bodyData
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function updatePersonalSpace(
		string $user,
		string $targetUser,
		array $bodyData,
	): ResponseInterface {
		$body = json_encode($bodyData, JSON_THROW_ON_ERROR);
		$space = $this->getPersonalSpace($targetUser);
		return GraphHelper::updateSpace(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$body,
			$space["id"],
			$this->featureContext->getStepLineRef()
		);
	}

	/**
	 * @When /^user "([^"]*)" (?:changes|tries to change) the name of the "([^"]*)" space to "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:changes|tries to change) the name of the "([^"]*)" space to "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $newName
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function updateSpaceName(
		string $user,
		string $spaceName,
		string $newName,
		string $owner = ''
	): void {
		$bodyData = ["Name" => $newName];
		$this->featureContext->setResponse(
			$this->updateSpace($user, $spaceName, $bodyData, $owner)
		);
	}

	/**
	 * @When /^user "([^"]*)" (?:changes|tries to change) the description of the "([^"]*)" space to "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:changes|tries to change) the description of the "([^"]*)" space to "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $newDescription
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function updateSpaceDescription(
		string $user,
		string $spaceName,
		string $newDescription,
		string $owner = ''
	): void {
		$bodyData = ["description" => $newDescription];
		$this->featureContext->setResponse(
			$this->updateSpace($user, $spaceName, $bodyData, $owner)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has changed the description of the "([^"]*)" space to "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $newDescription
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userHasChangedDescription(
		string $user,
		string $spaceName,
		string $newDescription
	): void {
		$bodyData = ["description" => $newDescription];
		$response = $this->updateSpace($user, $spaceName, $bodyData);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200",
			$response
		);
	}

	/**
	 * @When /^user "([^"]*)" (?:changes|tries to change) the quota of the "([^"]*)" space to "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:changes|tries to change) the quota of the "([^"]*)" space to "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param int $newQuota
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function updateSpaceQuota(
		string $user,
		string $spaceName,
		int $newQuota,
		string $owner = ''
	): void {
		$bodyData = ["quota" => ["total" => $newQuota]];
		$this->featureContext->setResponse(
			$this->updateSpace($user, $spaceName, $bodyData, $owner)
		);
	}

	/**
	 * @When user :user changes the quota of the personal space of user :targetUser to :newQuota using the Graph API
	 *
	 * @param string $user
	 * @param string $targetUser
	 * @param int $newQuota
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function updatePersonalSpaceQuota(
		string $user,
		string $targetUser,
		int $newQuota
	): void {
		$bodyData = ["quota" => ["total" => $newQuota]];
		$this->featureContext->setResponse(
			$this->updatePersonalSpace($user, $targetUser, $bodyData)
		);
	}

	/**
	 * @Given user :user has changed the quota of the personal space of user :targetUser to :newQuota
	 *
	 * @param string $user
	 * @param string $targetUser
	 * @param int $newQuota
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function userHasChangedTheQuotaOfTheSpaceTo(
		string $user,
		string $targetUser,
		int $newQuota
	): void {
		$bodyData = ["quota" => ["total" => $newQuota]];
		$response = $this->updatePersonalSpace($user, $targetUser, $bodyData);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200",
			$response
		);
	}

	/**
	 * @param string $user
	 * @param string $file
	 * @param string $type
	 * @param string $spaceName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function updateSpaceSpecialSection(
		string $user,
		string $file,
		string $type,
		string $spaceName
	): ResponseInterface {
		$space = $this->getSpaceByName($user, $spaceName);
		$spaceId = $space["id"];
		$fileId = $this->getFileId($user, $spaceName, $file);

		if ($type === "description") {
			$type = "readme";
		} else {
			$type = "image";
		}

		$bodyData = ["special" => [["specialFolder" => ["name" => "$type"], "id" => "$fileId"]]];
		$body = json_encode($bodyData, JSON_THROW_ON_ERROR);
		return GraphHelper::updateSpace(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$body,
			$spaceId,
			$this->featureContext->getStepLineRef()
		);
	}

	/**
	 * @When /^user "([^"]*)" sets the file "([^"]*)" as a (description|space image)\s? in a special section of the "([^"]*)" space$/
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $type
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function userSetsFileAsDescriptionOrImageInSpecialSectionOfSpace(
		string $user,
		string $file,
		string $type,
		string $spaceName
	): void {
		$this->featureContext->setResponse(
			$this->updateSpaceSpecialSection(
				$user,
				$file,
				$type,
				$spaceName
			)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has set the file "([^"]*)" as a (description|space image)\s? in a special section of the "([^"]*)" space$/
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $type
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function userHasUpdatedSpaceSpecialSection(
		string $user,
		string $file,
		string $type,
		string $spaceName
	): void {
		$response = $this->updateSpaceSpecialSection($user, $file, $type, $spaceName);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200",
			$response
		);
	}

	/**
	 * @Given /^user "([^"]*)" has created a space "([^"]*)" of type "([^"]*)" with quota "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string|null $spaceType
	 * @param int|null $quota
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasCreatedSpace(
		string $user,
		string $spaceName,
		string $spaceType,
		int $quota
	): void {
		$space = ["name" => $spaceName, "driveType" => $spaceType, "quota" => ["total" => $quota]];
		$response = $this->createSpace($user, $space);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			201,
			"Expected response status code should be 201 (Created)",
			$response
		);
	}

	/**
	 * @Given /^user "([^"]*)" has created a space "([^"]*)" with the default quota using the Graph API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserHasCreatedASpaceByDefaultUsingTheGraphApi(
		string $user,
		string $spaceName
	): void {
		$space = ["name" => $spaceName];
		$response = $this->createSpace($user, $space);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			201,
			"Expected response status code should be 201 (Created)",
			$response
		);
	}

	/**
	 * @param string $user
	 * @param array $space
	 * @param string $body
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function createSpace(
		string $user,
		array $space,
		string $body = ''
	): ResponseInterface {
		$body = $body !== '' ? $body : json_encode($space, JSON_THROW_ON_ERROR);
		$response = GraphHelper::createSpace(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$body,
			$this->featureContext->getStepLineRef()
		);
		if ($response->getStatusCode() === 201) {
			$this->addCreatedSpace($user, $response);
		}
		return $response;
	}

	/**
	 * @When /^user "([^"]*)" copies (?:file|folder) "([^"]*)" to "([^"]*)" inside space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fileDestination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userCopiesFileWithinSpaceUsingTheWebDAVAPI(
		string $user,
		string $fileSource,
		string $fileDestination,
		string $spaceName
	): void {
		$space = $this->getSpaceByName($user, $spaceName);
		$headers['Destination'] = $this->destinationHeaderValueWithSpaceName(
			$user,
			$fileDestination,
			$spaceName
		);

		$encodedName = \rawurlencode(ltrim($fileSource, "/"));
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebdavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $space["id"]);
		$fullUrl = "$baseUrl/$davPath/$encodedName";

		$this->featureContext->setResponse($this->copyFilesAndFoldersRequest($user, $fullUrl, $headers));
	}

	/**
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fileDestination
	 * @param string $spaceName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function moveFileWithinSpace(
		string $user,
		string $fileSource,
		string $fileDestination,
		string $spaceName
	): ResponseInterface {
		$space = $this->getSpaceByName($user, $spaceName);
		$headers['Destination'] = $this->destinationHeaderValueWithSpaceName(
			$user,
			$fileDestination,
			$spaceName
		);
		$headers['Overwrite'] = 'F';

		$encodedName = \rawurlencode(ltrim($fileSource, "/"));
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebdavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $space["id"]);
		$fullUrl = "$baseUrl/$davPath/$encodedName";

		return $this->moveFilesAndFoldersRequest($user, $fullUrl, $headers);
	}

	/**
	 * @When /^user "([^"]*)" moves (?:file|folder) "([^"]*)" to "([^"]*)" in space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fileDestination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userMovesFileWithinSpaceUsingTheWebDAVAPI(
		string $user,
		string $fileSource,
		string $fileDestination,
		string $spaceName
	): void {
		$this->featureContext->setResponse(
			$this->moveFileWithinSpace(
				$user,
				$fileSource,
				$fileDestination,
				$spaceName
			)
		);
		$this->featureContext->pushToLastHttpStatusCodesArray();
	}

	/**
	 * @Given /^user "([^"]*)" has moved (?:file|folder) "([^"]*)" to "([^"]*)" in space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fileDestination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasMovedFileWithinSpaceUsingTheWebDAVAPI(
		string $user,
		string $fileSource,
		string $fileDestination,
		string $spaceName
	): void {
		$response = $this->moveFileWithinSpace(
			$user,
			$fileSource,
			$fileDestination,
			$spaceName
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			201,
			__METHOD__ . "Expected response status code should be 201 (Created)\n" .
			"Actual response status code was: " . $response->getStatusCode() . "\n" .
			"Actual response body was: " . $response->getBody(),
			$response
		);
	}

	/**
	 * MOVE request for files|folders
	 *
	 * @param string $user
	 * @param string $fullUrl
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function moveFilesAndFoldersRequest(string $user, string $fullUrl, array $headers): ResponseInterface {
		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			'MOVE',
			$user,
			$this->featureContext->getPasswordForUser($user),
			$headers,
		);
	}

	/**
	 * @When /^user "([^"]*)" copies (?:file|folder) "([^"]*)" from space "([^"]*)" to "([^"]*)" inside space "([^"]*)" using the WebDAV API$/
	 * @When /^user "([^"]*)" copies (?:file|folder) "([^"]*)" from space "([^"]*)" to "([^"]*)" inside space "([^"]*)"(?: with following headers) using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fromSpaceName
	 * @param string $fileDestination
	 * @param string $toSpaceName
	 * @param TableNode|null $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userCopiesFileFromAndToSpaceBetweenSpaces(
		string $user,
		string $fileSource,
		string $fromSpaceName,
		string $fileDestination,
		string $toSpaceName,
		?TableNode $table = null
	): void {
		$space = $this->getSpaceByName($user, $fromSpaceName);
		$headers['Destination'] = $this->destinationHeaderValueWithSpaceName(
			$user,
			$fileDestination,
			$toSpaceName
		);

		if ($table !== null) {
			$this->featureContext->verifyTableNodeColumns(
				$table,
				['header', 'value']
			);
			foreach ($table as $row) {
				$headers[$row['header']] = $this->featureContext->substituteInLineCodes($row['value']);
			}
		}

		$encodedName = \rawurlencode(ltrim($fileSource, "/"));
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebdavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $space["id"]);
		$fullUrl = "$baseUrl/$davPath/$encodedName";

		$this->featureContext->setResponse($this->copyFilesAndFoldersRequest($user, $fullUrl, $headers));
	}

	/**
	 * @When /^user "([^"]*)" overwrites file "([^"]*)" from space "([^"]*)" to "([^"]*)" inside space "([^"]*)" while (copying|moving)\s? using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fromSpaceName
	 * @param string $fileDestination
	 * @param string $toSpaceName
	 * @param string $action
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userOverwritesFileFromAndToSpaceBetweenSpaces(
		string $user,
		string $fileSource,
		string $fromSpaceName,
		string $fileDestination,
		string $toSpaceName,
		string $action
	): void {
		$space = $this->getSpaceByName($user, $fromSpaceName);
		$headers['Destination'] = $this->destinationHeaderValueWithSpaceName(
			$user,
			$fileDestination,
			$toSpaceName
		);
		$headers['Overwrite'] = 'T';

		$encodedName = \rawurlencode(ltrim($fileSource, "/"));
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebdavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $space["id"]);
		$fullUrl = "$baseUrl/$davPath/$encodedName";

		if ($action === 'copying') {
			$response = $this->copyFilesAndFoldersRequest($user, $fullUrl, $headers);
		} else {
			$response = $this->moveFilesAndFoldersRequest($user, $fullUrl, $headers);
		}
		$this->featureContext->setResponse($response);
		$this->featureContext->pushToLastHttpStatusCodesArray();
	}

	/**
	 * @Then /^user "([^"]*)" (should|should not) be able to download file "([^"]*)" from space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $shouldOrNot
	 * @param string $fileName
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userShouldOrShouldNotBeAbleToDownloadFileFromSpace(
		string $user,
		string $shouldOrNot,
		string $fileName,
		string $spaceName
	): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$response = $this->featureContext->downloadFileAsUserUsingPassword(
			$user,
			$fileName,
			$this->featureContext->getPasswordForUser($user),
			null,
			$spaceId
		);
		if ($shouldOrNot === 'should') {
			$this->featureContext->theHTTPStatusCodeShouldBe(
				200,
				__METHOD__ . "Expected response status code is 200 but got " . $response->getStatusCode(),
				$response
			);
		} else {
			Assert::assertGreaterThanOrEqual(
				400,
				$response->getStatusCode(),
				__METHOD__
				. ' download must fail'
			);
			Assert::assertLessThanOrEqual(
				499,
				$response->getStatusCode(),
				__METHOD__
				. ' 4xx error expected but got status code "'
				. $response->getStatusCode() . '"'
			);
		}
	}

	/**
	 * @When /^user "([^"]*)" moves (?:file|folder) "([^"]*)" from space "([^"]*)" to "([^"]*)" inside space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fromSpaceName
	 * @param string $fileDestination
	 * @param string $toSpaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userMovesFileFromAndToSpaceBetweenSpaces(
		string $user,
		string $fileSource,
		string $fromSpaceName,
		string $fileDestination,
		string $toSpaceName
	): void {
		$space = $this->getSpaceByName($user, $fromSpaceName);
		$headers['Destination'] = $this->destinationHeaderValueWithSpaceName(
			$user,
			$fileDestination,
			$toSpaceName
		);

		$encodedName = \rawurlencode(ltrim($fileSource, "/"));
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebdavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $space["id"]);
		$fullUrl = "$baseUrl/$davPath/$encodedName";

		$this->featureContext->setResponse($this->moveFilesAndFoldersRequest($user, $fullUrl, $headers));
		$this->featureContext->pushToLastHttpStatusCodesArray();
	}

	/**
	 * returns a URL for destination with spacename
	 *
	 * @param string $user
	 * @param string $fileDestination
	 * @param string $spaceName
	 * @param string|null $endPath
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function destinationHeaderValueWithSpaceName(
		string $user,
		string $fileDestination,
		string $spaceName,
		?string $endPath = null
	): string {
		$space = $this->getSpaceByName($user, $spaceName);
		$fileDestination = $this->escapePath(\ltrim($fileDestination, "/"));
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebdavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $space["id"]);
		return "$baseUrl/$davPath/$fileDestination";
	}

	/**
	 * COPY request for files|folders
	 *
	 * @param string $user
	 * @param string $fullUrl
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function copyFilesAndFoldersRequest(string $user, string $fullUrl, array $headers): ResponseInterface {
		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			'COPY',
			$user,
			$this->featureContext->getPasswordForUser($user),
			$headers,
		);
	}

	/**
	 * @When /^user "([^"]*)" (copies|moves) file with id "([^"]*)" as "([^"]*)" into folder "([^"]*)" inside space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $actionType
	 * @param string $fileId
	 * @param string $destinationFile
	 * @param string $destinationFolder
	 * @param string $toSpaceName
	 *
	 * @throws GuzzleException
	 * @return void
	 */
	public function userCopiesOrMovesFileWithIdAsIntoFolderInsideSpace(
		string $user,
		string $actionType,
		string $fileId,
		string $destinationFile,
		string $destinationFolder,
		string $toSpaceName
	): void {
		$destinationFile = \trim($destinationFile, "/");
		$destinationFolder = \trim($destinationFolder, "/");
		$fileDestination = $destinationFolder . '/' . $this->escapePath($destinationFile);
		$baseUrl = $this->featureContext->getBaseUrl();
		$sourceDavPath = WebdavHelper::getDavPath($this->featureContext->getDavPathVersion());
		if ($toSpaceName === 'Shares') {
			$sharesPath = $this->featureContext->getSharesMountPath($user, $fileDestination);
			$davPath = WebDavHelper::getDavPath($this->featureContext->getDavPathVersion());
			$headers['Destination'] = "$baseUrl/$davPath/$sharesPath";
		} else {
			$headers['Destination'] = $this->destinationHeaderValueWithSpaceName(
				$user,
				$fileDestination,
				$toSpaceName,
				$fileId
			);
		}
		$fullUrl = "$baseUrl/$sourceDavPath/$fileId";
		if ($actionType === 'copies') {
			$this->featureContext->setResponse(
				$this->copyFilesAndFoldersRequest($user, $fullUrl, $headers)
			);
		} else {
			$this->featureContext->setResponse(
				$this->moveFilesAndFoldersRequest($user, $fullUrl, $headers)
			);
		}
	}

	/**
	 * @Given /^user "([^"]*)" renames file with id "([^"]*)" to "([^"]*)" inside space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $fileId
	 * @param string $destinationFile
	 * @param string $spaceName
	 *
	 * @throws GuzzleException
	 * @return void
	 */
	public function userRenamesFileWithIdToInsideSpace(
		string $user,
		string $fileId,
		string $destinationFile,
		string $spaceName
	): void {
		$destinationFile = \trim($destinationFile, "/");

		$fileDestination = $this->escapePath($destinationFile);

		$baseUrl = $this->featureContext->getBaseUrl();
		$sourceDavPath = WebdavHelper::getDavPath($this->featureContext->getDavPathVersion());
		if ($spaceName === 'Shares') {
			$sharesPath = $this->featureContext->getSharesMountPath($user, $fileDestination);
			$davPath = WebDavHelper::getDavPath($this->featureContext->getDavPathVersion());
			$headers['Destination'] = "$baseUrl/$davPath/$sharesPath";
		} else {
			$headers['Destination'] = $this->destinationHeaderValueWithSpaceName(
				$user,
				$fileDestination,
				$spaceName,
				$fileId
			);
		}
		$fullUrl = "$baseUrl/$sourceDavPath/$fileId";
		$this->featureContext->setResponse($this->moveFilesAndFoldersRequest($user, $fullUrl, $headers));
	}

	/**
	 * @Given /^user "([^"]*)" has (copied|moved) file with id "([^"]*)" as "([^"]*)" into folder "([^"]*)" inside space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $actionType
	 * @param string $fileId
	 * @param string $destinationFile
	 * @param string $destinationFolder
	 * @param string $spaceName
	 *
	 * @throws GuzzleException
	 * @return void
	 */
	public function userHasCopiedOrMovedFileInsideSpaceUsingFileId(
		string $user,
		string $actionType,
		string $fileId,
		string $destinationFile,
		string $destinationFolder,
		string $spaceName,
	): void {
		$destinationFile = \trim($destinationFile, "/");
		$destinationFolder = \trim($destinationFolder, "/");

		$fileDestination = $destinationFolder . '/' . $this->escapePath($destinationFile);

		$baseUrl = $this->featureContext->getBaseUrl();
		$sourceDavPath = WebdavHelper::getDavPath($this->featureContext->getDavPathVersion());
		if ($spaceName === 'Shares') {
			$sharesPath = $this->featureContext->getSharesMountPath($user, $fileDestination);
			$davPath = WebDavHelper::getDavPath($this->featureContext->getDavPathVersion());
			$headers['Destination'] = "$baseUrl/$davPath/$sharesPath";
		} else {
			$headers['Destination'] = $this->destinationHeaderValueWithSpaceName(
				$user,
				$fileDestination,
				$spaceName,
				$fileId
			);
		}
		$fullUrl = "$baseUrl/$sourceDavPath/$fileId";
		if ($actionType === 'copied') {
			$response = $this->copyFilesAndFoldersRequest($user, $fullUrl, $headers);
		} else {
			$response = $this->moveFilesAndFoldersRequest($user, $fullUrl, $headers);
		}
		Assert::assertEquals(
			201,
			$response->getStatusCode(),
			"Expected response status code should be 201"
		);
	}

	/**
	 * @Given /^user "([^"]*)" has renamed file with id "([^"]*)" to "([^"]*)" inside space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $fileId
	 * @param string $destinationFile
	 * @param string $spaceName
	 *
	 * @throws GuzzleException
	 * @return void
	 */
	public function userHasRenamedFileInsideSpaceUsingFileId(
		string $user,
		string $fileId,
		string $destinationFile,
		string $spaceName
	): void {
		$destinationFile = \trim($destinationFile, "/");

		$fileDestination = $this->escapePath($destinationFile);

		$baseUrl = $this->featureContext->getBaseUrl();
		$sourceDavPath = WebdavHelper::getDavPath($this->featureContext->getDavPathVersion());
		if ($spaceName === 'Shares') {
			$sharesPath = $this->featureContext->getSharesMountPath($user, $fileDestination);
			$davPath = WebDavHelper::getDavPath($this->featureContext->getDavPathVersion());
			$headers['Destination'] = "$baseUrl/$davPath/$sharesPath";
		} else {
			$headers['Destination'] = $this->destinationHeaderValueWithSpaceName(
				$user,
				$fileDestination,
				$spaceName,
				$fileId
			);
		}
		$fullUrl = "$baseUrl/$sourceDavPath/$fileId";
		$response = $this->moveFilesAndFoldersRequest($user, $fullUrl, $headers);
		Assert::assertEquals(
			201,
			$response->getStatusCode(),
			"Expected response status code should be 201"
		);
	}

	/**
	 * @When /^user "([^"]*)" tries to move (?:file|folder) "([^"]*)" of space "([^"]*)" to (space|folder) "([^"]*)" using its id in destination path$/
	 * @When /^user "([^"]*)" moves (?:file|folder) "([^"]*)" of space "([^"]*)" to (folder) "([^"]*)" using its id in destination path$/
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $sourceSpace
	 * @param string $destinationType
	 * @param string $destinationName
	 *
	 * @throws GuzzleException
	 * @return void
	 */
	public function userMovesFileToResourceUsingItsIdAsDestinationPath(
		string $user,
		string $source,
		string $sourceSpace,
		string $destinationType,
		string $destinationName
	): void {
		$source = \trim($source, "/");
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPathVersion = $this->featureContext->getDavPathVersion();
		$suffixPath = $user;
		if ($davPathVersion === WebDavHelper::DAV_VERSION_SPACES) {
			$suffixPath = $this->getSpaceIdByName($user, $sourceSpace);
		}
		$sourceDavPath = WebDavHelper::getDavPath($davPathVersion, $suffixPath);
		$fullUrl = "$baseUrl/$sourceDavPath/$source";

		if ($destinationType === "space") {
			$destinationId = $this->getSpaceIdByName($user, $destinationName);
		} else {
			$destinationId = $this->getResourceId($user, $sourceSpace, $destinationName);
		}
		$destinationDavPath = WebDavHelper::getDavPath($davPathVersion);
		$headers['Destination'] = "$baseUrl/$destinationDavPath/$destinationId";

		$response = $this->moveFilesAndFoldersRequest($user, $fullUrl, $headers);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @Given /^user "([^"]*)" has uploaded a file inside space "([^"]*)" with content "([^"]*)" to "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $fileContent
	 * @param string $destination
	 *
	 * @return array
	 * @throws GuzzleException
	 */
	public function userHasUploadedFile(
		string $user,
		string $spaceName,
		string $fileContent,
		string $destination
	): array {
		$response = $this->listAllAvailableSpacesOfUser($user);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$response = $this->featureContext->uploadFileWithContent(
			$user,
			$fileContent,
			$destination,
			$spaceId,
			true
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(['201', '204'], "", $response);
		return $response->getHeader('oc-fileid');
	}

	/**
	 * @When /^user "([^"]*)" shares a space "([^"]*)" with settings:$/
	 * @When /^user "([^"]*)" updates the space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function sendShareSpaceRequest(
		string $user,
		string $spaceName,
		TableNode $table
	): void {
		$rows = $table->getRowsHash();
		$this->featureContext->setResponse($this->shareSpace($user, $spaceName, $rows));
	}

	/**
	 * @param string $user
	 * @param string $spaceName
	 * @param array $rows
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public function shareSpace(string $user, string $spaceName, array $rows): ResponseInterface {
		$space = $this->getSpaceByName($user, $spaceName);
		$availableRoleToAssignToShareSpace = ['manager', 'editor', 'viewer'];
		if (isset($rows['role']) && !\in_array(\strtolower($rows['role']), $availableRoleToAssignToShareSpace)) {
			throw new Error("The Selected " . $rows['role'] . " Cannot be Found");
		}
		$rows["role"] = \array_key_exists("role", $rows) ? $rows["role"] : "viewer";
		$rows["shareType"] = \array_key_exists("shareType", $rows) ? $rows["shareType"] : 7;
		$rows["expireDate"] = \array_key_exists("expireDate", $rows) ? $rows["expireDate"] : null;

		$body = [
			"space_ref" => $space["id"],
			"shareType" => $rows["shareType"],
			"shareWith" => $rows["shareWith"],
			"role" => $rows["role"],
			"expireDate" => $rows["expireDate"]
		];

		$fullUrl = $this->featureContext->getBaseUrl() . $this->ocsApiUrl;

		return $this->sendPostRequestToUrl(
			$fullUrl,
			$user,
			$this->featureContext->getPasswordForUser($user),
			$body,
			$this->featureContext->getStepLineRef()
		);
	}

	/**
	 * @When /^user "([^"]*)" expires the (user|group) share of space "([^"]*)" for (?:user|group) "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $shareType
	 * @param  string $spaceName
	 * @param  string $memberUser
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userExpiresTheShareOfSpaceForUser(
		string $user,
		string $shareType,
		string $spaceName,
		string $memberUser
	) {
		$dateTime = new DateTime('yesterday');
		$rows['expireDate'] = $dateTime->format('Y-m-d\\TH:i:sP');
		$rows['shareWith'] = $memberUser;
		$rows['shareType'] = ($shareType === 'user') ? 7 : 8;
		$this->featureContext->setResponse($this->shareSpace($user, $spaceName, $rows));
	}

	/**
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode $table
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function createShareResource(
		string $user,
		string $spaceName,
		TableNode $table
	): ResponseInterface {
		$rows = $table->getRowsHash();
		$rows["path"] = \array_key_exists("path", $rows) ? $rows["path"] : null;
		$rows["shareType"] = \array_key_exists("shareType", $rows) ? $rows["shareType"] : 0;
		$rows["role"] = \array_key_exists("role", $rows) ? $rows["role"] : 'viewer';
		$rows["expireDate"] = \array_key_exists("expireDate", $rows) ? $rows["expireDate"] : null;

		$body = [
			"space_ref" => $this->getResourceId($user, $spaceName, $rows["path"]),
			"shareWith" => $rows["shareWith"],
			"shareType" => $rows["shareType"],
			"expireDate" => $rows["expireDate"],
			"role" => $rows["role"]
		];

		// share with custom permission
		if (isset($rows["permissions"])) {
			$body["permissions"] = $rows["permissions"];
		}

		$fullUrl = $this->featureContext->getBaseUrl() . $this->ocsApiUrl;
		$response = $this->sendPostRequestToUrl(
			$fullUrl,
			$user,
			$this->featureContext->getPasswordForUser($user),
			$body,
			$this->featureContext->getStepLineRef()
		);
		$responseXmlObject = HttpRequestHelper::getResponseXml($response, __METHOD__);
		$sharer = (string) $responseXmlObject->data->uid_owner;
		$this->featureContext->addToCreatedUserGroupshares($sharer, $responseXmlObject->data);
		return $response;
	}

	/**
	 * @When /^user "([^"]*)" creates a share inside of space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userCreatesShareInsideOfSpaceWithSettings(
		string $user,
		string $spaceName,
		TableNode $table
	): void {
		$this->featureContext->setResponse(
			$this->createShareResource($user, $spaceName, $table)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has created a share inside of space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasSharedTheFollowingEntityInsideOfSpace(
		string $user,
		string $spaceName,
		TableNode $table
	): void {
		$response = $this->createShareResource($user, $spaceName, $table);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
	}

	/**
	 * @When /^user "([^"]*)" changes the last share with settings:$/
	 *
	 * @param  string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function changeShareResourceWithSettings(
		string $user,
		TableNode $table
	): void {
		$rows = $table->getRowsHash();
		$this->featureContext->setResponse($this->updateSharedResource($user, $rows));
	}

	/**
	 * @param string $user
	 * @param array $rows
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function updateSharedResource(string $user, array $rows): ResponseInterface {
		$shareId = ($this->featureContext->isUsingSharingNG())
		? $this->featureContext->shareNgGetLastCreatedUserGroupShareID()
		: $this->featureContext->getLastCreatedUserGroupShareId();
		$fullUrl = $this->featureContext->getBaseUrl() . $this->ocsApiUrl . '/' . $shareId;
		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			"PUT",
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			null,
			$rows
		);
	}

	/**
	 * @When user :user expires the last share of resource :resource inside of the space :spaceName
	 *
	 * @param string $user
	 * @param string $resource
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException|JsonException
	 */
	public function userExpiresTheLastShareOfResourceInsideOfTheSpace(
		string $user,
		string $resource,
		string $spaceName
	): void {
		$dateTime = new DateTime('yesterday');
		$rows['expireDate'] = $dateTime->format('Y-m-d\\TH:i:sP');
		if ($this->featureContext->isUsingSharingNG()) {
			$space = $this->getSpaceByName($user, $spaceName);
			$itemId = $this->getResourceId($user, $spaceName, $resource);
			$body['expirationDateTime'] = $rows['expireDate'];
			$permissionID = $this->featureContext->shareNgGetLastCreatedUserGroupShareID();
			$this->featureContext->setResponse(
				GraphHelper::updateShare(
					$this->featureContext->getBaseUrl(),
					$this->featureContext->getStepLineRef(),
					$user,
					$this->featureContext->getPasswordForUser($user),
					$space["id"],
					$itemId,
					\json_encode($body),
					$permissionID
				)
			);
		} else {
			$rows['permissions'] = (string)$this->featureContext->getLastCreatedUserGroupShare()->permissions;
			$this->featureContext->setResponse($this->updateSharedResource($user, $rows));
		}
	}

	/**
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode $table
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function createPublicLinkToEntityInsideOfSpaceRequest(
		string $user,
		string $spaceName,
		TableNode $table
	): ResponseInterface {
		$space = $this->getSpaceByName($user, $spaceName);
		$rows = $table->getRowsHash();

		$rows["path"] = \array_key_exists("path", $rows) ? $rows["path"] : null;
		$rows["shareType"] = \array_key_exists("shareType", $rows) ? $rows["shareType"] : 3;
		$rows["permissions"] = \array_key_exists("permissions", $rows) ? $rows["permissions"] : null;
		$rows['password'] = \array_key_exists('password', $rows)
		? $this->featureContext->getActualPassword($rows['password']) : null;
		$rows["name"] = \array_key_exists("name", $rows) ? $rows["name"] : null;
		$rows["expireDate"] = \array_key_exists("expireDate", $rows) ? $rows["expireDate"] : null;

		$body = [
			"space_ref" => $space['id'] . "/" . $rows["path"],
			"shareType" => $rows["shareType"],
			"permissions" => $rows["permissions"],
			"password" => $rows["password"],
			"name" => $rows["name"],
			"expireDate" => $rows["expireDate"]
		];

		$fullUrl = $this->featureContext->getBaseUrl() . $this->ocsApiUrl;

		$response =  $this->sendPostRequestToUrl(
			$fullUrl,
			$user,
			$this->featureContext->getPasswordForUser($user),
			$body,
			$this->featureContext->getStepLineRef()
		);

		$responseXmlObject = HttpRequestHelper::getResponseXml($response, __METHOD__);
		$this->featureContext->addToCreatedPublicShares($responseXmlObject->data);
		return $response;
	}

	/**
	 * @When /^user "([^"]*)" creates a public link share inside of space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userCreatesPublicLinkShareInsideOfSpaceWithSettings(
		string $user,
		string $spaceName,
		TableNode $table
	): void {
		$this->featureContext->setResponse(
			$this->createPublicLinkToEntityInsideOfSpaceRequest($user, $spaceName, $table)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has created a public link share inside of space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasCreatedPublicLinkToEntityInsideOfSpaceRequest(
		string $user,
		string $spaceName,
		TableNode $table
	): void {
		$response = $this->createPublicLinkToEntityInsideOfSpaceRequest($user, $spaceName, $table);

		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200",
			$response
		);
	}

	/**
	 * @Given /^user "([^"]*)" has shared a space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  TableNode $tableNode
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasSharedSpace(
		string $user,
		string $spaceName,
		TableNode $tableNode
	): void {
		$rows = $tableNode->getRowsHash();
		$response = $this->shareSpace($user, $spaceName, $rows);
		$expectedHTTPStatus = "200";
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHTTPStatus,
			"Expected response status code should be $expectedHTTPStatus",
			$response
		);
		$expectedOCSStatus = "200";
		Assert::assertEquals(
			$expectedOCSStatus,
			$this->ocsContext->getOCSResponseStatusCode(
				$response
			),
			"Expected OCS response status code $expectedOCSStatus"
		);
	}

	/**
	 * @Given user :user has unshared a space :spaceName shared with :recipient
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  string $recipient
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasUnsharedASpaceSharedWith(
		string $user,
		string $spaceName,
		string $recipient
	): void {
		$response = $this->sendUnshareSpaceRequest($user, $spaceName, $recipient);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200",
			$response
		);
	}

	/**
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  string $recipient
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function sendUnshareSpaceRequest(
		string $user,
		string $spaceName,
		string $recipient
	): ResponseInterface {
		$space = $this->getSpaceByName($user, $spaceName);
		$fullUrl = $this->featureContext->getBaseUrl()
		. $this->ocsApiUrl . "/" . $space['id'] . "?shareWith=" . $recipient;

		return HttpRequestHelper::delete(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user)
		);
	}

	/**
	 * @When /^user "([^"]*)" unshares a space "([^"]*)" to (?:user|group) "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  string $recipient
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userUnsharesSpace(
		string $user,
		string $spaceName,
		string $recipient
	): void {
		$this->featureContext->setResponse(
			$this->sendUnshareSpaceRequest($user, $spaceName, $recipient)
		);
	}

	/**
	 * @param  string $user
	 * @param  string $object
	 * @param  string $spaceName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function removeObjectFromSpace(
		string $user,
		string $object,
		string $spaceName
	): ResponseInterface {
		$space = $this->getSpaceByName($user, $spaceName);

		$encodedName = \rawurlencode(ltrim($object, "/"));
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebdavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $space["id"]);
		$fullUrl = "$baseUrl/$davPath/$encodedName";

		return HttpRequestHelper::delete(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user)
		);
	}

	/**
	 * @When /^user "([^"]*)" removes the (?:file|folder) "([^"]*)" from space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $object
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userRemovesFileOrFolderFromSpace(
		string $user,
		string $object,
		string $spaceName
	): void {
		$this->featureContext->setResponse(
			$this->removeObjectFromSpace($user, $object, $spaceName)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has deleted a space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasDeletedASpaceOwnedByUser(
		string $user,
		string $spaceName
	): void {
		$response = $this->deleteSpace($user, $spaceName);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			204,
			"Expected response status code should be 200",
			$response
		);
	}

	/**
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  string $owner
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function disableSpace(
		string $user,
		string $spaceName,
		string $owner = ''
	): ResponseInterface {
		$space = $this->getSpaceByName(($owner !== "") ? $owner : $user, $spaceName);
		return GraphHelper::disableSpace(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$space["id"],
			$this->featureContext->getStepLineRef()
		);
	}

	/**
	 * @When /^user "([^"]*)" disables a space "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:disables|tries to disable) a space "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userDisablesSpace(
		string $user,
		string $spaceName,
		string $owner = ''
	): void {
		$this->featureContext->setResponse(
			$this->disableSpace($user, $spaceName, $owner)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has removed the (?:file|folder) "([^"]*)" from space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $object
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasRemovedFileOrFolderFromSpace(
		string $user,
		string $object,
		string $spaceName
	): void {
		$response = $this->removeObjectFromSpace($user, $object, $spaceName);
		$expectedHTTPStatus = "204";
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHTTPStatus,
			"Expected response status code should be $expectedHTTPStatus",
			$response
		);
	}

	/**
	 * @Given /^user "([^"]*)" has disabled a space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function sendUserHasDisabledSpaceRequest(
		string $user,
		string $spaceName
	): void {
		$response = $this->disableSpace($user, $spaceName);
		$expectedHTTPStatus = "204";
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHTTPStatus,
			"Expected response status code should be $expectedHTTPStatus",
			$response
		);
	}

	/**
	 * @param  string $user
	 * @param  string $spaceName
	 * @param string $owner
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function deleteSpace(
		string $user,
		string $spaceName,
		string $owner = ''
	): ResponseInterface {
		$space = $this->getSpaceByName(($owner !== "") ? $owner : $user, $spaceName);

		return GraphHelper::deleteSpace(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$space["id"],
			$this->featureContext->getStepLineRef()
		);
	}

	/**
	 * @When /^user "([^"]*)" deletes a space "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:deletes|tries to delete) a space "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userDeletesSpace(
		string $user,
		string $spaceName,
		string $owner = ''
	): void {
		$this->featureContext->setResponse(
			$this->deleteSpace($user, $spaceName, $owner)
		);
	}

	/**
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  string $owner
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function restoreSpace(
		string $user,
		string $spaceName,
		string $owner = ''
	): ResponseInterface {
		$space = $this->getSpaceByName(($owner !== "") ? $owner : $user, $spaceName);
		return GraphHelper::restoreSpace(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$space["id"],
			$this->featureContext->getStepLineRef()
		);
	}

	/**
	 * @When /^user "([^"]*)" restores a disabled space "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:restores|tries to restore) a disabled space "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userRestoresADisabledSpace(
		string $user,
		string $spaceName,
		string $owner = ''
	): void {
		$this->featureContext->setResponse(
			$this->restoreSpace($user, $spaceName, $owner)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has restored a disabled space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasRestoredSpaceRequest(
		string $user,
		string $spaceName
	): void {
		$response = $this->restoreSpace($user, $spaceName);
		$expectedHTTPStatus = "200";
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHTTPStatus,
			"Expected response status code should be $expectedHTTPStatus",
			$response
		);
	}

	/**
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function listAllDeletedFilesFromTrash(
		string $user,
		string $spaceName
	): ResponseInterface {
		$space = $this->getSpaceByName($user, $spaceName);
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebDavHelper::getDavPath(
			WebDavHelper::DAV_VERSION_SPACES,
			$space["id"],
			"trash-bin"
		);
		$fullUrl = "$baseUrl/$davPath";
		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			'PROPFIND',
			$user,
			$this->featureContext->getPasswordForUser($user)
		);
	}

	/**
	 * @When /^user "([^"]*)" lists all deleted files in the trash bin of the space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userListAllDeletedFilesInTrash(
		string $user,
		string $spaceName
	): void {
		$this->featureContext->setResponse(
			$this->listAllDeletedFilesFromTrash($user, $spaceName)
		);
	}

	/**
	 * @When /^user "([^"]*)" with admin permission lists all deleted files in the trash bin of the space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function adminListAllDeletedFilesInTrash(
		string $user,
		string $spaceName
	): void {
		// get space by admin user
		$space = $this->getSpaceByName($this->featureContext->getAdminUserName(), $spaceName);
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebDavHelper::getDavPath(
			WebDavHelper::DAV_VERSION_SPACES,
			$space["id"],
			"trash-bin"
		);
		$fullUrl = "$baseUrl/$davPath";
		$this->featureContext->setResponse(
			HttpRequestHelper::sendRequest(
				$fullUrl,
				$this->featureContext->getStepLineRef(),
				'PROPFIND',
				$user,
				$this->featureContext->getPasswordForUser($user)
			)
		);
	}

	/**
	 * User gets all objects in the trash of project space
	 *
	 * Method "getTrashbinContentFromResponseXml" returns array like:
	 * 	[1] => Array
	 *       (
	 *             [href] => /dav/spaces/trash-bin/spaceId/objectId/
	 *             [name] => deleted folder
	 *             [mtime] => 1649147272
	 *             [original-location] => deleted folder
	 *        )
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return array
	 * @throws GuzzleException
	 */
	public function getObjectsInTrashbin(
		string $user,
		string $spaceName
	): array {
		$response = $this->listAllDeletedFilesFromTrash($user, $spaceName);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			207,
			"Expected response status code should be 207",
			$response
		);
		return $this->trashbinContext->getTrashbinContentFromResponseXml(
			HttpRequestHelper::getResponseXml($response, __METHOD__)
		);
	}

	/**
	 * @Then /^as "([^"]*)" (?:file|folder|entry) "([^"]*)" should (not|)\s?exist in the trashbin of the space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $object
	 * @param  string $shouldOrNot   (not|)
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function checkExistenceOfObjectsInTrashbin(
		string $user,
		string $object,
		string $shouldOrNot,
		string $spaceName
	): void {
		$objectsInTrash = $this->getObjectsInTrashbin($user, $spaceName);

		$expectedObject = "";
		foreach ($objectsInTrash as $objectInTrash) {
			if ($objectInTrash["name"] === $object) {
				$expectedObject = $objectInTrash["name"];
			}
		}
		if ($shouldOrNot === "not") {
			Assert::assertEmpty($expectedObject, "$object is found in the trash, but should not be there");
		} else {
			Assert::assertNotEmpty($expectedObject, "$object is not found in the trash");
		}
	}

	/**
	 * @When /^user "([^"]*)" restores the (?:file|folder) "([^"]*)" from the trash of the space "([^"]*)" to "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $object
	 * @param string $spaceName
	 * @param string $destination
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function userRestoresSpaceObjectsFromTrashRequest(
		string $user,
		string $object,
		string $spaceName,
		string $destination
	): void {
		$space = $this->getSpaceByName($user, $spaceName);

		// find object in trash
		$objectsInTrash = $this->getObjectsInTrashbin($user, $spaceName);
		$pathToDeletedObject = "";
		foreach ($objectsInTrash as $objectInTrash) {
			if ($objectInTrash["name"] === $object) {
				$pathToDeletedObject = $objectInTrash["href"];
			}
		}

		if ($pathToDeletedObject === "") {
			throw new Exception(
				__METHOD__ . " Object '$object' was not found in the trashbin of space '$spaceName' by user '$user'"
			);
		}

		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebDavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $space["id"]);
		$destination = "$baseUrl/$davPath/$destination";
		$header = ["Destination" => $destination, "Overwrite" => "F"];

		$fullUrl = $baseUrl . $pathToDeletedObject;
		$this->featureContext->setResponse(
			HttpRequestHelper::sendRequest(
				$fullUrl,
				$this->featureContext->getStepLineRef(),
				'MOVE',
				$user,
				$this->featureContext->getPasswordForUser($user),
				$header,
				""
			)
		);
	}

	/**
	 * @When user :user deletes the file/folder :resource from the trash of the space :spaceName
	 * @When user :user tries to delete the file/folder :resource from the trash of the space :spaceName
	 *
	 * @param string $user
	 * @param string $object
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function userDeletesObjectsFromTrashRequest(
		string $user,
		string $object,
		string $spaceName
	): void {
		// find object in trash
		$objectsInTrash = $this->getObjectsInTrashbin($user, $spaceName);
		$pathToDeletedObject = "";
		foreach ($objectsInTrash as $objectInTrash) {
			if ($objectInTrash["name"] === $object) {
				$pathToDeletedObject = $objectInTrash["href"];
			}
		}

		if ($pathToDeletedObject === "") {
			throw new Exception(
				__METHOD__ . " Object '$object' was not found in the trashbin of space '$spaceName' by user '$user'"
			);
		}

		$fullUrl = $this->featureContext->getBaseUrl() . $pathToDeletedObject;
		$this->featureContext->setResponse(
			HttpRequestHelper::sendRequest(
				$fullUrl,
				$this->featureContext->getStepLineRef(),
				'DELETE',
				$user,
				$this->featureContext->getPasswordForUser($user),
				[],
				""
			)
		);
	}

	/**
	 * User downloads a preview of the file inside the project space
	 *
	 * @When /^user "([^"]*)" downloads the preview of "([^"]*)" of the space "([^"]*)" with width "([^"]*)" and height "([^"]*)" using the WebDAV API$/
	 *
	 * @param  string $user
	 * @param  string $fileName
	 * @param  string $spaceName
	 * @param  string $width
	 * @param  string $height
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function downloadPreview(
		string $user,
		string $fileName,
		string $spaceName,
		string $width,
		string $height
	): void {
		$eTag = str_replace("\"", "", $this->getETag($user, $spaceName, $fileName));
		$urlParameters = [
			'scalingup' => 0,
			'preview' => '1',
			'a' => '1',
			'c' => $eTag,
			'x' => $width,
			'y' => $height
		];
		$urlParameters = \http_build_query($urlParameters, '', '&');
		$space = $this->getSpaceByName($user, $spaceName);

		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebDavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $space["id"]);
		$fullUrl = "$baseUrl/$davPath/$fileName?$urlParameters";
		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$fullUrl,
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user)
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" downloads the file "([^"]*)" of the space "([^"]*)" using the WebDAV API$/
	 *
	 * @param  string $user
	 * @param  string $fileName
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function downloadFile(
		string $user,
		string $fileName,
		string $spaceName
	): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$response = $this->featureContext->downloadFileAsUserUsingPassword(
			$user,
			$fileName,
			$this->featureContext->getPasswordForUser($user),
			[],
			$spaceId
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When /^user "([^"]*)" requests the checksum of (?:file|folder|entry) "([^"]*)" in space "([^"]*)" via propfind using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userRequestsTheChecksumViaPropfindInSpace(
		string $user,
		string $path,
		string $spaceName
	): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$this->featureContext->setResponse(
			$this->checksumContext->propfindResourceChecksum($user, $path, $spaceId)
		);
	}

	/**
	 * @When /^user "([^"]*)" uploads file with checksum "([^"]*)" and content "([^"]*)" to "([^"]*)" in space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $checksum
	 * @param string $content
	 * @param string $destination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userUploadsFileWithChecksumWithContentInSpace(
		string $user,
		string $checksum,
		string $content,
		string $destination,
		string $spaceName
	): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$this->featureContext->setResponse(
			$this->featureContext->uploadFileWithChecksumAndContent(
				$user,
				$checksum,
				$content,
				$destination,
				false,
				$spaceId
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" downloads version of the file "([^"]*)" with the index "([^"]*)" of the space "([^"]*)" using the WebDAV API$/
	 * @When /^user "([^"]*)" tries to download version of the file "([^"]*)" with the index "([^"]*)" of the space "([^"]*)" using the WebDAV API$/
	 *
	 * @param  string $user
	 * @param  string $fileName
	 * @param  string $index
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function downloadVersionOfTheFile(
		string $user,
		string $fileName,
		string $index,
		string $spaceName
	): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$this->featureContext->setResponse(
			$this->filesVersionsContext->downloadVersion($user, $fileName, $index, $spaceId)
		);
	}

	/**
	 * @When user :user tries to get versions of the file :file from the space :space using the WebDAV API
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userTriesToDownloadFileVersions(string $user, string $file, string $spaceName): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$this->featureContext->setResponse(
			$this->filesVersionsContext->getFileVersions($user, $file, null, $spaceId)
		);
	}

	/**
	 * return the etag for an element inside a space
	 *
	 * @param string $user requestor
	 * @param string $space space name
	 * @param string $path path to the file
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function userGetsEtagOfElementInASpace(string $user, string $space, string $path): string {
		$spaceId = $this->getSpaceIdByName($user, $space);
		$xmlObject = $this->webDavPropertiesContext->storeEtagOfElement($user, $path, '', $spaceId);
		return $this->featureContext->getEtagFromResponseXmlObject($xmlObject);
	}

	/**
	 * saves the etag of an element in a space
	 *
	 * @param string $user requestor
	 * @param string $space space name
	 * @param string $path path to the file
	 * @param ?string $storePath path to the file in the store
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function storeEtagOfElementInSpaceForUser(
		string $user,
		string $space,
		string $path,
		?string $storePath = ""
	): void {
		if ($storePath === "") {
			$storePath = $path;
		}
		$this->storedEtags[$user][$space][$storePath]
			= $this->userGetsEtagOfElementInASpace(
				$user,
				$space,
				$path,
			);
	}

	/**
	 * returns stored etag for an element if present
	 *
	 * @param string $user
	 * @param string $space
	 * @param string $path
	 *
	 * @return string
	 */
	public function getStoredEtagForPathInSpaceOfAUser(string $user, string $space, string $path): string {
		Assert::assertArrayHasKey(
			$user,
			$this->storedEtags,
			__METHOD__ . " No stored etags for user '$user' found"
			. "\nFound: " . print_r($this->storedEtags, true)
		);
		Assert::assertArrayHasKey(
			$space,
			$this->storedEtags[$user],
			__METHOD__ . " No stored etags for user '$user' with space '$space' found"
			. "\nFound: " . implode(', ', array_keys($this->storedEtags[$user]))
		);
		Assert::assertArrayHasKey(
			$path,
			$this->storedEtags[$user][$space],
			__METHOD__ . " No stored etags for user '$user' with space '$space' with path '$path' found"
			. '\nFound: ' . print_r($this->storedEtags[$user][$space], true)
		);
		return $this->storedEtags[$user][$space][$path];
	}

	/**
	 * @Then /^these etags (should|should not) have changed$/
	 *
	 * @param string $action
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function theseEtagsShouldShouldNotHaveChanged(string $action, TableNode $table): void {
		$this->featureContext->verifyTableNodeColumns($table, ["user", "path", "space"]);
		$this->featureContext->verifyTableNodeColumnsCount($table, 3);
		$changedEtagCount = 0;
		$changedEtagMessage = __METHOD__;
		foreach ($table->getColumnsHash() as $row) {
			$user = $row['user'];
			$path = $row['path'];
			$space = $row['space'];
			$etag = $this->userGetsEtagOfElementInASpace($user, $space, $path);
			$storedEtag = $this->getStoredEtagForPathInSpaceOfAUser($user, $space, $path);
			if ($action === 'should' && $etag === $storedEtag) {
				$changedEtagCount++;
				$changedEtagMessage .=
				"\nExpected etag of element '$path' for  user '$user' in space '$space' to change, but it did not.";
			}
			if ($action === 'should not' && $etag !== $storedEtag) {
				$changedEtagCount++;
				$changedEtagMessage .=
				"\nExpected etag of element '$path' for  user '$user' in space '$space' to change, but it did not.";
			}
		}

		Assert::assertEquals(0, $changedEtagCount, $changedEtagMessage);
	}

	/**
	 * @Given /^user "([^"]*)" has stored etag of element "([^"]*)" inside space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $space
	 *
	 * @return void
	 * @throws GuzzleException | Exception
	 */
	public function userHasStoredEtagOfElementFromSpace(string $user, string $path, string $space): void {
		$user = $this->featureContext->getActualUsername($user);
		$this->storeEtagOfElementInSpaceForUser(
			$user,
			$space,
			$path,
		);
		if ($this->storedEtags[$user][$space][$path] === "" || $this->storedEtags[$user][$space][$path] === null) {
			throw new Exception("Expected stored etag to be some string but found null!");
		}
	}

	/**
	 * @Given /^user "([^"]*)" has stored etag of element "([^"]*)" on path "([^"]*)" inside space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $storePath
	 * @param string $space
	 *
	 * @return void
	 * @throws Exception | GuzzleException
	 */
	public function userHasStoredEtagOfElementOnPathFromSpace(
		string $user,
		string $path,
		string $storePath,
		string $space
	): void {
		$user = $this->featureContext->getActualUsername($user);
		$this->storeEtagOfElementInSpaceForUser(
			$user,
			$space,
			$path,
			$storePath
		);
		if ($this->storedEtags[$user][$space][$storePath] === ""
			|| $this->storedEtags[$user][$space][$storePath] === null
		) {
			throw new Exception("Expected stored etag to be some string but found null!");
		}
	}

	/**
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode|null $table
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function sendShareSpaceViaLinkRequest(
		string $user,
		string $spaceName,
		?TableNode $table
	): ResponseInterface {
		$space = $this->getSpaceByName($user, $spaceName);
		$rows = $table->getRowsHash();

		$rows["shareType"] = \array_key_exists("shareType", $rows) ? $rows["shareType"] : 3;
		$rows["permissions"] = \array_key_exists("permissions", $rows) ? $rows["permissions"] : null;
		$rows['password'] = \array_key_exists('password', $rows)
		? $this->featureContext->getActualPassword($rows['password']) : null;
		$rows["name"] = \array_key_exists("name", $rows) ? $rows["name"] : null;
		$rows["expireDate"] = \array_key_exists("expireDate", $rows) ? $rows["expireDate"] : null;

		$body = [
			"space_ref" => $space['id'],
			"shareType" => $rows["shareType"],
			"permissions" => $rows["permissions"],
			"password" => $rows["password"],
			"name" => $rows["name"],
			"expireDate" => $rows["expireDate"]
		];

		$fullUrl = $this->featureContext->getBaseUrl() . $this->ocsApiUrl;

		$response = $this->sendPostRequestToUrl(
			$fullUrl,
			$user,
			$this->featureContext->getPasswordForUser($user),
			$body,
			$this->featureContext->getStepLineRef()
		);

		$this->featureContext->addToCreatedPublicShares(
			HttpRequestHelper::getResponseXml($response, __METHOD__)->data
		);
		return $response;
	}

	/**
	 * @When /^user "([^"]*)" creates a public link share of the space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode|null $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userCreatesAPublicLinkShareOfSpaceWithSettings(
		string $user,
		string $spaceName,
		?TableNode $table
	): void {
		$this->featureContext->setResponse(
			$this->sendShareSpaceViaLinkRequest(
				$user,
				$spaceName,
				$table
			)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has created a public link share of the space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode|null $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasCreatedPublicLinkShareOfSpace(
		string $user,
		string $spaceName,
		?TableNode $table
	): void {
		$response = $this->sendShareSpaceViaLinkRequest($user, $spaceName, $table);

		$expectedHTTPStatus = "200";
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHTTPStatus,
			"Expected response status code should be $expectedHTTPStatus",
			$response
		);
	}

	/**
	 * @Then /^for user "([^"]*)" the space "([^"]*)" should (not|)\s?contain the last created public link$/
	 * @Then /^for user "([^"]*)" the space "([^"]*)" should (not|)\s?contain the last created (public link|share) of the file "([^"]*)"$/
	 *
	 * @param string    $user
	 * @param string    $spaceName
	 * @param string    $shouldOrNot   (not|)
	 * @param string    $shareType
	 * @param string    $fileName
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function forUserSpaceShouldContainLinks(
		string $user,
		string $spaceName,
		string $shouldOrNot,
		string $shareType = 'public link',
		string $fileName = ''
	): void {
		if (!empty($fileName)) {
			$body = $this->getFileId($user, $spaceName, $fileName);
		} else {
			$space = $this->getSpaceByName($user, $spaceName);
			$body = $space['id'];
		}

		$url = "/apps/files_sharing/api/v1/shares?reshares=true&space_ref=" . $body;

		$response = $this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			'GET',
			$url,
		);

		$should = ($shouldOrNot !== "not");
		$responseArray = json_decode(
			json_encode(HttpRequestHelper::getResponseXml($response, __METHOD__)->data),
			true,
			512,
			JSON_THROW_ON_ERROR
		);

		if ($should) {
			Assert::assertNotEmpty($responseArray, __METHOD__ . ' Response should contain a link, but it is empty');
			foreach ($responseArray as $element) {
				if ($shareType === 'public link') {
					$expectedLinkId = ($this->featureContext->isUsingSharingNG())
					? $this->featureContext->shareNgGetLastCreatedLinkShareID() :
					 (string) $this->featureContext->getLastCreatedPublicShare()->id;
					Assert::assertEquals($element["id"], $expectedLinkId, "link IDs are different");
				} else {
					$expectedShareId = ($this->featureContext->isUsingSharingNG())
					? $this->featureContext->shareNgGetLastCreatedUserGroupShareID()
					: (string)$this->featureContext->getLastCreatedUserGroupShareId();
					Assert::assertEquals($element["id"], $expectedShareId, "share IDs are different");
				}
			}
		} else {
			Assert::assertEmpty($responseArray, __METHOD__ . ' Response should be empty');
		}
	}

	/**
	 * @When /^user "([^"]*)" gets the following properties of (?:file|folder|entry|resource) "([^"]*)" inside space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string    $user
	 * @param string    $resourceName
	 * @param string    $spaceName
	 * @param TableNode|null $propertiesTable
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function userGetsTheFollowingPropertiesOfFileInsideSpaceUsingTheWebdavApi(
		string $user,
		string $resourceName,
		string $spaceName,
		TableNode $propertiesTable
	): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$response = $this->webDavPropertiesContext->getPropertiesOfFolder(
			$user,
			$resourceName,
			$spaceId,
			$propertiesTable
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When /^user "([^"]*)" gets the following extracted properties of (?:file|folder|entry|resource) "([^"]*)" inside space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string    $user
	 * @param string    $resourceName
	 * @param string    $spaceName
	 * @param TableNode|null $propertiesTable
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function userGetsTheFollowingExtractedPropertiesOfFileInsideSpaceUsingTheWebdavApi(
		string $user,
		string $resourceName,
		string $spaceName,
		TableNode $propertiesTable
	): void {
		// NOTE: extracting properties occurs asynchronously
		// short wait is necessary before getting those properties
		sleep(2);
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$response = $this->webDavPropertiesContext->getPropertiesOfFolder(
			$user,
			$resourceName,
			$spaceId,
			$propertiesTable
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @Then /^as user "([^"]*)" (?:file|folder|entry|resource) "([^"]*)" inside space "([^"]*)" should contain a property "([^"]*)" with value "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $resourceName
	 * @param string $spaceName
	 * @param string $property
	 * @param string $value
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function userGetsTheFollowingPropertiesOfFileInsideSpaceWithValueUsingTheWebdavApi(
		string $user,
		string $resourceName,
		string $spaceName,
		string $property,
		string $value
	): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$this->webDavPropertiesContext->checkPropertyOfAFolder(
			$user,
			$resourceName,
			$property,
			$value,
			null,
			$spaceId
		);
	}

	/**
	 * @Then /^as user "([^"]*)" (?:file|folder|entry) "([^"]*)" inside space "([^"]*)" (should|should not) be favorited$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $spaceName
	 * @param string $shouldOrNot
	 *
	 * @return void
	 */
	public function asUserFileOrFolderInsideSpaceShouldOrNotBeFavorited(
		string $user,
		string $path,
		string $spaceName,
		string $shouldOrNot
	): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$this->favoritesContext->asUserFileOrFolderShouldBeFavorited(
			$user,
			$path,
			($shouldOrNot === 'should') ? 1 : 0,
			$spaceId
		);
	}

	/**
	 * @When /^user "([^"]*)" favorites element "([^"]*)" in space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userFavoritesElementInSpaceUsingTheWebdavApi(string $user, string $path, string $spaceName): void {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$this->featureContext->setResponse($this->favoritesContext->userFavoritesElement($user, $path, $spaceId));
	}

	/**
	 * @Given /^user "([^"]*)" has stored id of (?:file|folder) "([^"]*)" of the space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function userHasStoredIdOfPathOfTheSpace(string $user, string $path, string $spaceName): void {
		$this->getSpaceIdByName($user, $spaceName);
		$this->featureContext->setStoredFileID($this->featureContext->getFileIdForPath($user, $path));
	}

	/**
	 * @Then /^user "([^"]*)" (folder|file) "([^"]*)" of the space "([^"]*)" should have the previously stored id$/
	 *
	 * @param string $user
	 * @param string $fileOrFolder
	 * @param string $path
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userFolderOfTheSpaceShouldHaveThePreviouslyStoredId(
		string $user,
		string $fileOrFolder,
		string $path,
		string $spaceName
	): void {
		$this->getSpaceIdByName($user, $spaceName);
		$user = $this->featureContext->getActualUsername($user);
		$currentFileID = $this->featureContext->getFileIdForPath($user, $path);
		$storedFileID = $this->featureContext->getStoredFileID();
		Assert::assertEquals(
			$currentFileID,
			$storedFileID,
			__METHOD__
			. " User '$user' $fileOrFolder '$path' does not have the previously stored id '"
			. $storedFileID . "', but has '$currentFileID'."
		);
	}

	/**
	 * @Then /^for user "([^"]*)" the search result should contain space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function searchResultShouldContainSpace(string $user, string $spaceName): void {
		$responseArray = json_decode(
			json_encode(
				HttpRequestHelper::getResponseXml($this->featureContext->getResponse())->xpath("//d:response/d:href")
			),
			true,
			512,
			JSON_THROW_ON_ERROR
		);
		Assert::assertNotEmpty($responseArray, "search result is empty");

		// for mountpoint, id looks a little different than for project space
		if (str_contains($spaceName, 'mountpoint')) {
			$splitSpaceName = explode("/", $spaceName);
			$space = $this->getSpaceByName($user, $splitSpaceName[1]);
			$splitSpaceId = explode("$", $space['id']);
			$spaceId = str_replace('!', '%21', $splitSpaceId[1]);
		} else {
			$space = $this->getSpaceByName($user, $spaceName);
			$spaceId = $space['id'];
		}
		$suffixPath = $user;
		$davPathVersion = $this->featureContext->getDavPathVersion();
		if ($davPathVersion === WebDavHelper::DAV_VERSION_SPACES) {
			$suffixPath = $spaceId;
		}

		$topWebDavPath = "/" . WebDavHelper::getDavPath($davPathVersion, $suffixPath);

		$spaceFound = false;
		foreach ($responseArray as $value) {
			if ($topWebDavPath === $value[0]) {
				$spaceFound = true;
			}
		}
		Assert::assertTrue($spaceFound, "response does not contain the space '$spaceName'");
	}

	/**
	 * @When /^user "([^"]*)" sends PROPFIND request to space "([^"]*)" using the WebDAV API$/
	 * @When /^user "([^"]*)" sends PROPFIND request to space "([^"]*)" with depth "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param ?string $folderDepth
	 *
	 * @return void
	 *
	 * @throws JsonException
	 *
	 * @throws GuzzleException
	 */
	public function userSendsPropfindRequestToSpaceUsingTheWebdavApi(
		string $user,
		string $spaceName,
		?string $folderDepth = "1"
	): void {
		$this->featureContext->setResponse(
			$this->sendPropfindRequestToSpace($user, $spaceName, "", null, $folderDepth)
		);
	}

	/**
	 * @When /^user "([^"]*)" sends PROPFIND request from the space "([^"]*)" to the resource "([^"]*)" with depth "([^"]*)" using the WebDAV API$/
	 * @When user :user sends PROPFIND request from the space :spaceName to the resource :resource using the WebDAV API
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $resource
	 * @param string|null $folderDepth
	 *
	 * @return void
	 *
	 * @throws JsonException
	 *
	 * @throws GuzzleException
	 */
	public function userSendsPropfindRequestFromTheSpaceToTheResourceWithDepthUsingTheWebdavApi(
		string $user,
		string $spaceName,
		string $resource,
		?string $folderDepth = "1"
	): void {
		$response = $this->sendPropfindRequestToSpace($user, $spaceName, $resource, null, $folderDepth);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When /^user "([^"]*)" sends PROPFIND request to space "([^"]*)" with headers using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param TableNode $headersTable
	 *
	 * @return void
	 *
	 * @throws JsonException
	 *
	 * @throws GuzzleException
	 */
	public function userSendsPropfindRequestToSpaceWithHeaders(
		string $user,
		string $spaceName,
		TableNode $headersTable
	): void {
		$this->featureContext->verifyTableNodeColumns(
			$headersTable,
			['header', 'value']
		);
		$headers = [];
		foreach ($headersTable as $row) {
			$headers[$row['header']] = $row ['value'];
		}
		$this->featureContext->setResponse(
			$this->sendPropfindRequestToSpace($user, $spaceName, '', $headers, '0')
		);
	}

	/**
	 * @param string $user
	 * @param string $spaceName
	 * @param string|null $resource
	 * @param array|null $headers
	 * @param string|null $folderDepth
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 *
	 * @throws JsonException
	 */
	public function sendPropfindRequestToSpace(
		string $user,
		string $spaceName,
		?string $resource = "",
		?array $headers = [],
		?string $folderDepth = "1"
	): ResponseInterface {
		$spaceId = $this->getSpaceIdByName($user, $spaceName);
		$properties = [
			'oc:id',
			'oc:fileid',
			'oc:spaceid',
			'oc:file-parent',
			'oc:shareid',
			'oc:name',
			'd:displayname',
			'd:getetag',
			'oc:permissions',
			'd:resourcetype',
			'oc:size',
			'd:getlastmodified',
			'oc:tags',
			'oc:favorite',
			'oc:share-types',
			'oc:privatelink',
			'd:getcontenttype',
			'd:lockdiscovery',
			'd:activelock'
		];

		$davPathVersion = $this->featureContext->getDavPathVersion();
		if ($spaceName === 'Shares' && $davPathVersion !== WebDavHelper::DAV_VERSION_SPACES) {
			// for old/new dav paths, append the Shares space path
			if ($resource === '' || $resource === '/') {
				$resource = $spaceName;
			} else {
				$resource = "$spaceName/$resource";
			}
		}

		return WebDavHelper::propfind(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			$resource,
			$properties,
			$this->featureContext->getStepLineRef(),
			$folderDepth,
			$spaceId,
			"files",
			$davPathVersion,
			null,
			$headers
		);
	}

	/**
	 * @Then /^as user "([^"]*)" the (?:PROPFIND|REPORT) response should contain a (resource|space) "([^"]*)" with these key and value pairs:$/
	 *
	 * @param string $user
	 * @param string $type	# type should be either resource or space
	 * @param string $resource
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function asUsertheXMLResponseShouldContainMountpointWithTheseKeyAndValuePair(
		string $user,
		string $type,
		string $resource,
		TableNode $table
	): void {
		$this->featureContext->verifyTableNodeColumns($table, ['key', 'value']);
		if ($this->featureContext->getDavPathVersion() === WebDavHelper::DAV_VERSION_SPACES && $type === 'space') {
			$space = $this->getSpaceByName($user, $resource);
			$resource = $space['id'];
		} elseif (\preg_match(GraphHelper::jsonSchemaRegexToPureRegex(GraphHelper::getFileIdRegex()), $resource)) {
			// When using file-id, some characters need to be encoded
			$resource = \str_replace("!", "%21", $resource);
		} else {
			$encodedPaths = \array_map(fn ($path) => \rawurlencode($path), \explode("/", $resource));
			$resource = \join("/", $encodedPaths);
		}
		$this->theXMLResponseShouldContain($resource, $table->getHash());
	}

	/**
	 * @param SimpleXMLElement $responseXmlObject
	 * @param array $xpaths
	 * @param string $message
	 *
	 * @return string
	 */
	public function buildXpathErrorMessage(
		SimpleXMLElement $responseXmlObject,
		array $xpaths,
		string $message
	): string {
		return "Using xpaths:\n\t- " . \join("\n\t- ", $xpaths)
			. "\n"
			. $message
			. "\n\t"
			. "'" . \trim($responseXmlObject->asXML()) . "'";
	}

	/**
	 * @param SimpleXMLElement $responseXmlObject
	 * @param string $siblingXpath
	 * @param string $siblingToFind
	 *
	 * @return string
	 * @throws Exception
	 */
	public function getXpathSiblingValue(
		SimpleXMLElement $responseXmlObject,
		string $siblingXpath,
		string $siblingToFind
	): string {
		$xpaths[] = $siblingXpath . "/preceding-sibling::$siblingToFind";
		$xpaths[] = $siblingXpath . "/following-sibling::$siblingToFind";

		foreach ($xpaths as $key => $xpath) {
			$foundSibling = $responseXmlObject->xpath($xpath);
			if (\count($foundSibling)) {
				break;
			}
		}
		$errorMessage = $this->buildXpathErrorMessage(
			$responseXmlObject,
			$xpaths,
			"Could not find sibling '<$siblingToFind>' element in the XML response"
		);
		Assert::assertNotEmpty($foundSibling, $errorMessage);
		return \preg_quote($foundSibling[0]->__toString(), "/");
	}

	/**
	 * @param string $resource	// can be resource name, space id or file id
	 * @param array $properties	// ["key" => "value"]
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theXMLResponseShouldContain(string $resource, array $properties): void {
		$responseXmlObject = HttpRequestHelper::getResponseXml($this->featureContext->getResponse(), __METHOD__);
		$hrefs = array_map(fn ($href) => $href->__toString(), $responseXmlObject->xpath("//d:response/d:href"));

		$currentHref = '';
		foreach ($hrefs as $href) {
			if (\str_ends_with(\rtrim($href, "/"), "/$resource")) {
				$currentHref = $href;
				break;
			}
		}

		foreach ($properties as $property) {
			$itemToFind = $property['key'];

			// if href is not found, build xpath using oc:name
			$xpaths = [];
			if (!$currentHref) {
				$decodedResource = \urldecode($resource);
				if ($property['key'] === 'oc:name') {
					$xpath = "//oc:name[text()='$decodedResource']";
				} elseif (\array_key_exists('oc:shareroot', $properties)) {
					$xpaths[] = "//oc:name[text()='$resource']/preceding-sibling::oc:shareroot[text()='"
					. $properties['oc:shareroot'] . "'/preceding-sibling::/";
					$xpaths[] = "//oc:name[text()='$resource']/preceding-sibling::oc:shareroot[text()='"
					. $properties['oc:shareroot'] . "'/following-sibling::/";
					$xpaths[] = "//oc:name[text()='$resource']/following-sibling::oc:shareroot[text()='"

					. $properties['oc:shareroot'] . "'/preceding-sibling::/";
					$xpaths[] = "//oc:name[text()='$resource']/following-sibling::oc:shareroot[text()='"
					 . $properties['oc:shareroot'] . "'/following-sibling::/";
				} else {
					$xpaths[] = "//oc:name[text()='$decodedResource']/preceding-sibling::";
					$xpaths[] = "//oc:name[text()='$decodedResource']/following-sibling::";
				}
			} else {
				$xpath = "//d:href[text()='$currentHref']/following-sibling::d:propstat//$itemToFind";
			}

			if (\count($xpaths)) {
				// check every xpath
				foreach ($xpaths as $key => $path) {
					$xpath = "{$path}{$itemToFind}";
					$foundXmlItem = $responseXmlObject->xpath($xpath);
					$xpaths[$key] = $xpath;
					if (\count($foundXmlItem)) {
						break;
					}
				}
			} else {
				$foundXmlItem = $responseXmlObject->xpath($xpath);
				$xpaths[] = $xpath;
			}

			Assert::assertCount(
				1,
				$foundXmlItem,
				$this->buildXpathErrorMessage(
					$responseXmlObject,
					$xpaths,
					"Found multiple elements for '<$itemToFind>' in the XML response"
				)
			);
			Assert::assertNotEmpty(
				$foundXmlItem,
				$this->buildXpathErrorMessage(
					$responseXmlObject,
					$xpaths,
					"Could not find '<$itemToFind>' element in the XML response"
				)
			);

			$actualValue = $foundXmlItem[0]->__toString();
			$expectedValue = $property['value'];
			\preg_match_all("/%self::[a-z0-9-:]+?%/", $expectedValue, $selfMatches);
			$substituteFunctions = [];
			if (!empty($selfMatches[0])) {
				$siblingXpath = $xpaths[\count($xpaths) - 1];
				foreach ($selfMatches[0] as $match) {
					$siblingToFind = \ltrim($match, "/%self::/");
					$siblingToFind = \rtrim($siblingToFind, "/%/");
					$substituteFunctions[] = [
						"code" => $match,
						"function" =>
							[$this, "getXpathSiblingValue"],
						"parameter" => [$responseXmlObject, $siblingXpath, $siblingToFind]
					];
				}
			}
			$expectedValue = $this->featureContext->substituteInLineCodes(
				$property['value'],
				null,
				[],
				$substituteFunctions,
			);

			switch ($itemToFind) {
				case "oc:fileid":
					$expectedValue = GraphHelper::jsonSchemaRegexToPureRegex($expectedValue);
					Assert::assertMatchesRegularExpression(
						$expectedValue,
						$actualValue,
						'wrong "fileid" in the response'
					);
					break;
				case "oc:file-parent":
					$expectedValue = GraphHelper::jsonSchemaRegexToPureRegex($expectedValue);
					Assert::assertMatchesRegularExpression(
						$expectedValue,
						$actualValue,
						'wrong "file-parent" in the response'
					);
					break;
				case "oc:privatelink":
					$expectedValue = GraphHelper::jsonSchemaRegexToPureRegex($expectedValue);
					Assert::assertMatchesRegularExpression(
						$expectedValue,
						$actualValue,
						'wrong "privatelink" in the response'
					);
					break;
				case "oc:tags":
					// The value should be a comma-separated string of tag names.
					// We do not care what order they happen to be in, so compare as sorted lists.
					$expectedTags = \explode(",", $expectedValue);
					\sort($expectedTags);
					$expectedTags = \implode(",", $expectedTags);

					$actualTags = \explode(",", $actualValue);
					\sort($actualTags);
					$actualTags = \implode(",", $actualTags);
					Assert::assertEquals($expectedTags, $actualTags, "wrong '$itemToFind' in the response");
					break;
				case "d:lockdiscovery/d:activelock/d:timeout":
					if ($expectedValue === "Infinity") {
						Assert::assertEquals($expectedValue, $actualValue, "wrong '$itemToFind' in the response");
					} else {
						// some time may be required between a lock and propfind request.
						$responseValue = explode('-', $actualValue);
						$responseValue = \intval($responseValue[1]);
						$expectedValue = explode('-', $expectedValue);
						$expectedValue = \intval($expectedValue[1]);
						Assert::assertTrue($responseValue >= ($expectedValue - 3));
					}
					break;
				case "oc:remote-item-id":
					$expectedValue = GraphHelper::jsonSchemaRegexToPureRegex($expectedValue);
					Assert::assertMatchesRegularExpression(
						$expectedValue,
						$actualValue,
						'wrong "remote-item-id" in the response'
					);
					break;
				default:
					Assert::assertEquals($expectedValue, $actualValue, "wrong '$itemToFind' in the response");
					break;
			}
		}
	}

	/**
	 * @Then as user :user the key :key from PROPFIND response should match with shared-with-me drive-item-id of share :resource
	 *
	 * @param string $user
	 * @param string $key
	 * @param string $resource
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function asUserTheKeyFromPropfindResponseShouldMatchWithSharedwithmeDriveitemidOfShare(
		string $user,
		string $key,
		string $resource
	): void {
		$responseXmlObject = HttpRequestHelper::getResponseXml($this->featureContext->getResponse(), __METHOD__);
		$fileId = $responseXmlObject->xpath("//oc:name[text()='$resource']/preceding-sibling::$key")[0]->__toString();

		$jsonResponse = GraphHelper::getSharesSharedWithMe(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user)
		);
		$driveItemId = '';
		$jsonResponseBody = $this->featureContext->getJsonDecodedResponseBodyContent($jsonResponse);
		foreach ($jsonResponseBody->value as $value) {
			if ($value->name === "$resource") {
				$driveItemId = $value->id;
				break;
			} else {
				throw new Error("Response didn't contain a share $resource");
			}
		}
		Assert::assertEquals($fileId, $driveItemId, "File-id '$fileId' doesn't match driveItemId '$driveItemId'");
	}

	/**
	 * @When /^public downloads the folder "([^"]*)" from the last created public link using the public files API$/
	 *
	 * @param string $resource
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function publicDownloadsTheFolderFromTheLastCreatedPublicLink(string $resource) {
		$token = ($this->featureContext->isUsingSharingNG())
		? $this->featureContext->shareNgGetLastCreatedLinkShareToken()
		: $this->featureContext->getLastCreatedPublicShareToken();

		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$this->featureContext->listFolder(
				$token,
				$resource,
				'0',
				['oc:fileid'],
				null,
				"public-files"
			)
		);
		$resourceId = $responseXmlObject->xpath("//d:response/d:propstat/d:prop/oc:fileid");
		$queryString = 'public-token=' . $token . '&id=' . $resourceId[0][0];
		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$this->archiverContext->getArchiverUrl($queryString),
				$this->featureContext->getStepLineRef(),
				'',
				''
			)
		);
	}

	/**
	 * @Then the relative quota amount should be :quota_amount
	 *
	 * @param string $quotaAmount
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theRelativeQuotaAmountShouldBe(string $quotaAmount): void {
		$data = $this->ocsContext->getOCSResponseData($this->featureContext->getResponse());
		if (isset($data->quota, $data->quota->relative)) {
			Assert::assertEquals(
				$data->quota->relative,
				$quotaAmount,
				"Expected relative quota amount to be $quotaAmount but found to be $data->quota->relative"
			);
			return;
		}
		Assert::fail("No relative quota amount found in response");
	}

	/**
	 * @Then /^the user "([^"]*)" should have a space called "([^"]*)" granted to (user|group)\s? "([^"]*)" with role "([^"]*)"$/
	 * @Then /^the user "([^"]*)" should have a space called "([^"]*)" granted to (user|group)\s? "([^"]*)" with role "([^"]*)" and expiration date "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $recipientType
	 * @param string $recipient
	 * @param string $role
	 * @param string|null $expirationDate
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function theUserShouldHaveSpaceWithRecipient(
		string $user,
		string $spaceName,
		string $recipientType,
		string $recipient,
		string $role,
		?string $expirationDate = null
	): void {
		$response = $this->listAllAvailableSpacesOfUser($user, '$expand=root($expand=permissions)');
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200",
			$response
		);
		Assert::assertIsArray(
			$spaceAsArray = $this->getSpaceByNameFromResponse($spaceName, $response),
			"No space with name $spaceName found"
		);
		$recipientType === 'user' ?
		$recipientId = $this->featureContext->getAttributeOfCreatedUser($recipient, 'id')
		: $recipientId = $this->featureContext->getGroupIdByGroupName($recipient);
		$foundRoleInResponse = false;
		foreach ($spaceAsArray['root']['permissions'] as $permission) {
			if (isset($permission['grantedToIdentities'][0][$recipientType])
				&& $permission['roles'][0] === $role
				&& $permission['grantedToIdentities'][0][$recipientType]['id'] === $recipientId
			) {
				$foundRoleInResponse = true;
				if ($expirationDate !== null && isset($permission['expirationDateTime'])) {
					Assert::assertEquals(
						$expirationDate,
						(preg_split("/[\sT]+/", $permission['expirationDateTime']))[0],
						"$expirationDate is different in the response"
					);
				}
				break;
			}
		}
		Assert::assertTrue($foundRoleInResponse, "the response does not contain the $recipientType $recipient");
	}

	/**
	 * @When user :user tries to download the space :spaceName owned by user :owner using the WebDAV API
	 * @When /^user "([^"]*)" (?:downloads|tries to download) the space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $owner
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function userDownloadsTheSpaceUsingTheWebdavApi(string $user, string $spaceName, string $owner = ''): void {
		$space = $this->getSpaceByName($owner ?: $user, $spaceName);
		$url = $this->archiverContext->getArchiverUrl('id=' . $space['id']);
		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$url,
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user),
			)
		);
	}

	/**
	 * @Given user :sharer has shared resource :path inside space :space with user :sharee
	 *
	 * @param $sharer string
	 * @param $path string
	 * @param $space string
	 * @param $sharee string
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasSharedResourceInsideSpaceWithUser(
		string $sharer,
		string $path,
		string $space,
		string $sharee
	): void {
		$sharer = $this->featureContext->getActualUsername($sharer);
		$resource_id = $this->getResourceId($sharer, $space, $path);
		$response = $this->featureContext->createShare(
			$sharer,
			$path,
			'0',
			$this->featureContext->getActualUsername($sharee),
			null,
			null,
			null,
			null,
			null,
			$resource_id
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
		$responseStatusCode = $this->ocsContext->getOCSResponseStatusCode(
			$response
		);
		$statusCodes = ["100", "200"];
		Assert::assertContainsEquals(
			$responseStatusCode,
			$statusCodes,
			"OCS status code is not any of the expected values "
			. \implode(",", $statusCodes) . " got " . $responseStatusCode
		);
	}

	/**
	 * @When user :user gets the file :file from space :space using the Graph API
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $space
	 *
	 * @return void
	 */
	public function userGetsTheDriveItemInSpace(string $user, string $file, string $space): void {
		$spaceId = ($this->getSpaceByName($user, $space))["id"];
		$itemId = '';
		if ($space === "Shares") {
			$itemId = GraphHelper::getShareMountId(
				$this->featureContext->getBaseUrl(),
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$file
			);
		} else {
			$itemId = $this->getFileId($user, $space, $file);
		}
		$url = $this->featureContext->getBaseUrl() . "/graph/v1.0/drives/$spaceId/items/$itemId";
		// NOTE: extracting properties occurs asynchronously
		// short wait is necessary before getting those properties
		sleep(2);
		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$url,
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user),
			)
		);
	}
}
