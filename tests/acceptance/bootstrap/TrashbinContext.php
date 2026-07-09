<?php declare(strict_types=1);
/**
 * @author Vincent Petry <pvince81@owncloud.com>
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

use Behat\Behat\Context\Context;
use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use Behat\Gherkin\Node\TableNode;
use GuzzleHttp\Exception\GuzzleException;
use PHPUnit\Framework\Assert;
use Psr\Http\Message\ResponseInterface;
use TestHelpers\HttpRequestHelper;
use TestHelpers\WebDavHelper;
use TestHelpers\BehatHelper;
use Behat\Step\Given;
use Behat\Step\Then;
use Behat\Step\When;

require_once 'bootstrap.php';

/**
 * Trashbin context
 */
class TrashbinContext implements Context {
	private FeatureContext $featureContext;

	/**
	 * @param string|null $user user
	 *
	 * @return ResponseInterface
	 */
	public function emptyTrashbin(?string $user): ResponseInterface {
		$user = $this->featureContext->getActualUsername($user);
		$davPathVersion = $this->featureContext->getDavPathVersion();
		return WebDavHelper::makeDavRequest(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			'DELETE',
			null,
			[],
			null,
			$this->featureContext->getStepLineRef(),
			null,
			$davPathVersion,
			'trash-bin'
		);
	}

	/**
	 *
	 * @param string $user user
	 *
	 * @return void
	 */
	#[When('user :user empties the trashbin using the trashbin API')]
	public function userEmptiesTrashbin(string $user): void {
		$this->featureContext->setResponse($this->emptyTrashbin($user));
	}

	/**
	 * Get files list from the response from trashbin api
	 *
	 * @param SimpleXMLElement|null $responseXmlObject
	 *
	 * @return array
	 */
	public function getTrashbinContentFromResponseXml(?SimpleXMLElement $responseXmlObject): array {
		$xmlElements = $responseXmlObject->xpath('//d:response');
		$files = \array_map(
			static function (SimpleXMLElement $element) {
				$href = $element->xpath('./d:href')[0];

				$propStats = $element->xpath('./d:propstat');
				$successPropStat = \array_filter(
					$propStats,
					static function (SimpleXMLElement $propStat) {
						$status = $propStat->xpath('./d:status');
						return (string) $status[0] === 'HTTP/1.1 200 OK';
					}
				);
				if (isset($successPropStat[0])) {
					$successPropStat = $successPropStat[0];

					$name = $successPropStat->xpath('./d:prop/oc:trashbin-original-filename');
					$mtime = $successPropStat->xpath('./d:prop/oc:trashbin-delete-timestamp');
					$resourcetype = $successPropStat->xpath('./d:prop/d:resourcetype');
					if (\array_key_exists(0, $resourcetype)
						&& ($resourcetype[0]->asXML() === "<d:resourcetype><d:collection/></d:resourcetype>")
					) {
						$collection[0] = true;
					} else {
						$collection[0] = false;
					}
					$originalLocation = $successPropStat->xpath('./d:prop/oc:trashbin-original-location');
				} else {
					$name = [];
					$mtime = [];
					$collection = [];
					$originalLocation = [];
				}

				return [
					'href' => (string) $href,
					'name' => isset($name[0]) ? (string) $name[0] : null,
					'mtime' => isset($mtime[0]) ? (string) $mtime[0] : null,
					'collection' => $collection[0] ?? false,
					'original-location' => isset($originalLocation[0]) ? (string) $originalLocation[0] : null
				];
			},
			$xmlElements
		);

		return $files;
	}

	/**
	 * List the top of the trashbin folder for a user
	 *
	 * @param string|null $user user
	 * @param string $depth
	 *
	 * @return array response
	 * @throws Exception
	 */
	public function listTopOfTrashbinFolder(?string $user, string $depth = "1"): array {
		$password = $this->featureContext->getPasswordForUser($user);
		$davPathVersion = $this->featureContext->getDavPathVersion();

		$suffixPath = $user;
		$spaceId = null;
		if ($davPathVersion === WebDavHelper::DAV_VERSION_SPACES) {
			$spaceId = WebDavHelper::getPersonalSpaceIdForUser(
				$this->featureContext->getBaseUrl(),
				$user,
				$password,
				$this->featureContext->getStepLineRef()
			);
			$suffixPath = $spaceId;
		}
		$response = WebDavHelper::listFolder(
			$this->featureContext->getBaseUrl(),
			$user,
			$password,
			"",
			$depth,
			$spaceId,
			$this->featureContext->getStepLineRef(),
			[
				'oc:trashbin-original-filename',
				'oc:trashbin-original-location',
				'oc:trashbin-delete-timestamp',
				'd:getlastmodified'
			],
			'trash-bin',
			$davPathVersion
		);
		$this->featureContext->setResponse($response);

		$files = $this->getTrashbinContentFromResponseXml(
			HttpRequestHelper::getResponseXml(
				$response,
				__METHOD__
			)
		);
		// filter root element
		$files = \array_filter(
			$files,
			static function ($element) use ($davPathVersion, $suffixPath) {
				$davPath = WebDavHelper::getDavPath($davPathVersion, $suffixPath, "trash-bin");
				return ($element['href'] !== "/" . $davPath . "/");
			}
		);
		return $files;
	}

	/**
	 * List trashbin folder
	 *
	 * @param string|null $user user
	 * @param string $depth
	 *
	 * @return array of all the items in the trashbin of the user
	 * @throws Exception
	 */
	public function listTrashbinFolder(?string $user, string $depth = "1"): array {
		return $this->listTrashbinFolderCollection(
			$user,
			"",
			$depth
		);
	}

	/**
	 * List a collection in the trashbin
	 *
	 * @param string|null $user user
	 * @param string|null $collectionPath the string of ids of the folder and sub-folders
	 * @param string $depth
	 * @param int $level
	 *
	 * @return array response
	 * @throws Exception
	 */
	public function listTrashbinFolderCollection(
		?string $user,
		?string $collectionPath = "",
		string $depth = "1",
		int $level = 1
	): array {
		// $collectionPath should be some list of file-ids like 2147497661/2147497662
		// or the empty string, which will list the whole trashbin from the top.
		$collectionPath = \trim($collectionPath, "/");
		$password = $this->featureContext->getPasswordForUser($user);
		$davPathVersion = $this->featureContext->getDavPathVersion();
		$response = WebDavHelper::listFolder(
			$this->featureContext->getBaseUrl(),
			$user,
			$password,
			$collectionPath,
			$depth,
			null,
			$this->featureContext->getStepLineRef(),
			[
				'oc:trashbin-original-filename',
				'oc:trashbin-original-location',
				'oc:trashbin-delete-timestamp',
				'd:resourcetype',
				'd:getlastmodified'
			],
			'trash-bin',
			$davPathVersion
		);
		$response->getBody()->rewind();
		$statusCode = $response->getStatusCode();
		$respBody = $response->getBody()->getContents();
		Assert::assertEquals(
			"207",
			$statusCode,
			"Expected status code to be '207' but got $statusCode \nResponse\n$respBody"
		);

		$files = $this->getTrashbinContentFromResponseXml(
			HttpRequestHelper::getResponseXml(
				$response,
				__METHOD__ . " $collectionPath"
			)
		);

		$suffixPath = $user;
		if ($davPathVersion === WebDavHelper::DAV_VERSION_SPACES) {
			$suffixPath = WebDavHelper::getPersonalSpaceIdForUser(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$this->featureContext->getStepLineRef()
			);
		}
		$endpoint = WebDavHelper::getDavPath($davPathVersion, $suffixPath, "trash-bin");

		// filter out the collection itself, we only want to return the members
		$files = \array_filter(
			$files,
			static function ($element) use ($endpoint, $collectionPath) {
				$path = $collectionPath;
				if ($path !== "") {
					$path = $path . "/";
				}
				return ($element['href'] !== "/$endpoint/$path");
			}
		);

		foreach ($files as $file) {
			// check for unexpected/invalid href values and fail early in order to
			// avoid "common" situations that could cause infinite recursion.
			$trashbinRef = $file["href"];
			$trimmedTrashbinRef = \trim($trashbinRef, "/");
			$expectedStartLength = \strlen($endpoint);
			if ((\substr($trimmedTrashbinRef, 0, $expectedStartLength) !== $endpoint)
				|| (\strlen($trimmedTrashbinRef) === $expectedStartLength)
			) {
				// A top href (maybe without even the username) has been returned
				// in the response. That should never happen, or have been filtered out
				// by the code above.
				throw new Exception(
					__METHOD__ . " Error: unexpected href in trashbin propfind at level $level: '$trashbinRef'"
				);
			}
			if ($file["collection"]) {
				$trimmedHref = \trim($trashbinRef, "/");
				$explodedHref = \explode("/", $trimmedHref);
				$trashbinId = $collectionPath . "/" . end($explodedHref);
				$nextFiles = $this->listTrashbinFolderCollection(
					$user,
					$trashbinId,
					$depth,
					$level + 1
				);
				// Filter the collection element. We only want the members.
				$nextFiles = \array_filter(
					$nextFiles,
					static function ($element) use ($user, $trashbinRef) {
						return ($element['href'] !== $trashbinRef);
					}
				);
				\array_push($files, ...$nextFiles);
			}
		}
		return $files;
	}

	/**
	 *
	 * @param string $user
	 * @param string $depth
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :user lists the resources in the trashbin with depth :depth using the WebDAV API')]
	public function userGetsFilesInTheTrashbinWithDepthUsingTheWebdavApi(string $user, string $depth): void {
		$this->listTopOfTrashbinFolder($user, $depth);
	}

	/**
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the trashbin DAV response should not contain these nodes')]
	public function theTrashbinDavResponseShouldNotContainTheseNodes(TableNode $table): void {
		$this->featureContext->verifyTableNodeColumns($table, ['name']);
		$files = $this->getTrashbinContentFromResponseXml(
			HttpRequestHelper::getResponseXml($this->featureContext->getResponse(), __METHOD__)
		);

		foreach ($table->getHash() as $row) {
			$path = trim((string)$row['name'], "/");
			foreach ($files as $file) {
				if (trim((string)$file['original-location'], "/") === $path) {
					throw new Exception("file $path was not expected in trashbin response but was found");
				}
			}
		}
	}

	/**
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the trashbin DAV response should contain these nodes')]
	public function theTrashbinDavResponseShouldContainTheseNodes(TableNode $table): void {
		$this->featureContext->verifyTableNodeColumns($table, ['name']);

		$files = $this->getTrashbinContentFromResponseXml(
			HttpRequestHelper::getResponseXml($this->featureContext->getResponse(), __METHOD__)
		);

		foreach ($table->getHash() as $row) {
			$path = trim($row['name'], "/");
			$found = false;
			foreach ($files as $file) {
				if (trim((string)$file['original-location'], "/") === $path) {
					$found = true;
					break;
				}
			}
			if (!$found) {
				throw new Exception("file $path was expected in trashbin response but was not found");
			}
		}
	}

	/**
	 * Send a webdav request to list the trashbin content
	 *
	 * @param string $user user
	 * @param string|null $asUser - To send request as another user
	 * @param string|null $password
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function sendTrashbinListRequest(
		string $user,
		?string $asUser = null,
		?string $password = null
	): ResponseInterface {
		$asUser = $asUser ?? $user;
		$password = $password ?? $this->featureContext->getPasswordForUser($asUser);

		return WebDavHelper::propfind(
			$this->featureContext->getBaseUrl(),
			$user,
			$password,
			null,
			[
				'oc:trashbin-original-filename',
				'oc:trashbin-original-location',
				'oc:trashbin-delete-timestamp',
				'd:getlastmodified'
			],
			$this->featureContext->getStepLineRef(),
			'1',
			null,
			'trash-bin',
			$this->featureContext->getDavPathVersion(),
			$asUser
		);
	}

	/**
	 *
	 * @param string $asUser
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :asUser tries to list the trashbin content for user :user')]
	public function userTriesToListTheTrashbinContentForUser(string $asUser, string $user) {
		$user = $this->featureContext->getActualUsername($user);
		$asUser = $this->featureContext->getActualUsername($asUser);
		$response = $this->sendTrashbinListRequest($user, $asUser);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $asUser
	 * @param string $user
	 * @param string $password
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :asUser tries to list the trashbin content for user :user using password :password')]
	public function userTriesToListTheTrashbinContentForUserUsingPassword(
		string $asUser,
		string $user,
		string $password
	): void {
		$response = $this->sendTrashbinListRequest($user, $asUser, $password);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param TableNode $elements
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the last webdav response should not contain the following elements')]
	public function theLastWebdavResponseShouldNotContainFollowingElements(TableNode $elements): void {
		$files = $this->getTrashbinContentFromResponseXml(
			HttpRequestHelper::getResponseXml($this->featureContext->getResponse())
		);

		// 'user' is also allowed in the table even though it is not used anywhere
		// This for better readability in feature files
		$this->featureContext->verifyTableNodeColumns($elements, ['path'], ['path', 'user']);
		$elementRows = $elements->getHash();
		foreach ($elementRows as $expectedElement) {
			$notFound = true;
			$expectedPath = "/" . \ltrim($expectedElement['path'], "/");
			foreach ($files as $file) {
				// Allow the table of expected elements to have entries that do
				// not have to specify the "implied" leading slash, or have multiple
				// leading slashes, to make scenario outlines more flexible
				if ($expectedPath === $file['original-location']) {
					$notFound = false;
				}
			}
			Assert::assertTrue($notFound, "$expectedPath expected not to be listed in response but found");
		}
	}

	/**
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $ofUser
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :user tries to delete the file with original path :path from the trashbin of user :ofUser using the trashbin API')]
	public function userTriesToDeleteFromTrashbinOfUser(string $user, string $path, string $ofUser): void {
		$response = $this->deleteItemFromTrashbin($user, $path, $ofUser);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $ofUser
	 * @param string $password
	 *
	 * @return void
	 * @throws JsonException
	 * @throws Exception
	 */
	#[When('user :user tries to delete the file with original path :path from the trashbin of user :ofUser using the password :password and the trashbin API')]
	public function userTriesToDeleteFromTrashbinOfUserUsingPassword(
		string $user,
		string $path,
		string $ofUser,
		string $password
	): void {
		$response = $this->deleteItemFromTrashbin($user, $path, $ofUser, $password);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string|null $asUser
	 * @param string|null $path
	 * @param string|null $user
	 *
	 * @return void
	 * @throws JsonException
	 * @throws Exception
	 */
	#[When('user :asUser tries to restore the file with original path :path from the trashbin of user :user using the trashbin API')]
	public function userTriesToRestoreFromTrashbinOfUser(?string $asUser, ?string $path, ?string $user): void {
		$user = $this->featureContext->getActualUsername($user);
		$asUser = $this->featureContext->getActualUsername($asUser);
		$response = $this->restoreElement($user, $path, null, $asUser);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string|null $asUser
	 * @param string|null $path
	 * @param string|null $user
	 * @param string|null $password
	 *
	 * @return void
	 * @throws JsonException
	 * @throws Exception
	 */
	#[When('user :asUser tries to restore the file with original path :path from the trashbin of user :user using the password :password and the trashbin API')]
	public function userTriesToRestoreFromTrashbinOfUserUsingPassword(
		?string $asUser,
		?string $path,
		?string $user,
		?string $password
	): void {
		$asUser = $this->featureContext->getActualUsername($asUser);
		$user = $this->featureContext->getActualUsername($user);
		$response = $this->restoreElement($user, $path, null, $asUser, $password);
		$this->featureContext->setResponse($response);
	}

	/**
	 * converts the trashItemHRef from /<base>/dav/trash-bin/<user>/<item_id>/ to /trash-bin/<user>/<item_id>
	 *
	 * @param string $href
	 *
	 * @return string
	 */
	private function convertTrashbinHref(string $href): string {
		$trashItemHRef = \trim($href, '/');
		$trashItemHRef = \strstr($trashItemHRef, '/trash-bin');
		$trashItemHRef = \trim($trashItemHRef, '/');
		$parts = \explode('/', $trashItemHRef);
		$decodedParts = \array_slice($parts, 2);
		return '/' . \join('/', $decodedParts);
	}

	/**
	 * @param string $user
	 * @param string $originalPath
	 * @param string|null $ofUser
	 * @param string|null $password
	 *
	 * @return ResponseInterface
	 */
	public function deleteItemFromTrashbin(
		string $user,
		string $originalPath,
		?string $ofUser = null,
		?string $password = null
	): ResponseInterface {
		$ofUser = $ofUser ?? $user;
		$user = $this->featureContext->getActualUsername($user);
		$ofUser = $this->featureContext->getActualUsername($ofUser);

		$listing = $this->listTrashbinFolder($ofUser);

		$path = "";
		$originalPath = \trim($originalPath, '/');
		foreach ($listing as $entry) {
			// The entry for the trashbin root can have original-location null.
			// That is reasonable, because the trashbin root is not something that can be restored.
			$originalLocation = $entry['original-location'] ?? '';
			if (\trim($originalLocation, '/') === $originalPath) {
				$path = $entry['href'];
				break;
			}
		}

		if ($path === "") {
			throw new Exception(
				__METHOD__
				. " could not find the trashbin entry for original path '$originalPath' of user '$user'"
			);
		}

		$password = $password ?? $this->featureContext->getPasswordForUser($user);
		$fullUrl = $this->featureContext->getBaseUrl() . $path;

		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			"DELETE",
			$user,
			$password
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $originalPath
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('/^user "([^"]*)" deletes the (?:file|folder|entry) with original path "([^"]*)" from the trashbin using the trashbin API$/')]
	public function deleteFileFromTrashbin(string $user, string $originalPath): void {
		$response = $this->deleteItemFromTrashbin($user, $originalPath);
		$this->featureContext->setResponse($response);
		$this->featureContext->pushToLastStatusCodesArrays();
	}

	/**
	 *
	 * @param string $user
	 * @param string $originalPath
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Given('/^user "([^"]*)" has deleted the (?:file|folder|entry) with original path "([^"]*)" from the trashbin$/')]
	public function userHasDeletedTheFolderWithOriginalPathFromTheTrashbin(string $user, string $originalPath): void {
		$response = $this->deleteItemFromTrashbin($user, $originalPath);
		$this->featureContext->theHTTPStatusCodeShouldBe(204, '', $response);
	}

	/**
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('/^user "([^"]*)" deletes the following (?:files|folders|entries) with original path from the trashbin$/')]
	public function deleteFollowingFilesFromTrashbin(string $user, TableNode $table): void {
		$this->featureContext->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		foreach ($paths as $path) {
			$response = $this->deleteItemFromTrashbin($user, $path["path"]);
			$this->featureContext->setResponse($response);
			$this->featureContext->pushToLastStatusCodesArrays();
		}
	}

	/**
	 *
	 * @param string|null $user
	 * @param string|null $path
	 *
	 * @return void
	 * @throws JsonException
	 * @throws Exception
	 */
	#[Then('/^as "([^"]*)" (?:file|folder|entry) "([^"]*)" should exist in the trashbin$/')]
	public function asFileOrFolderExistsInTrash(?string $user, ?string $path): void {
		$user = $this->featureContext->getActualUsername($user);
		$path = \trim($path, '/');
		$sections = \explode('/', $path, 2);

		$firstEntry = $this->findFirstTrashedEntry($user, \trim($sections[0], '/'));

		Assert::assertNotNull(
			$firstEntry,
			"The first trash entry was not found while looking for trashbin entry '$path' of user '$user'"
		);

		if (\count($sections) !== 1) {
			$listing = $this->listTrashbinFolderCollection($user, \basename(\rtrim($firstEntry['href'], '/')));
		} else {
			$listing = [];
		}

		// query was on the main element?
		if (\count($sections) === 1) {
			// already found, return
			return;
		}

		$checkedName = \basename($path);

		$found = false;
		foreach ($listing as $entry) {
			if ($entry['name'] === $checkedName) {
				$found = true;
				break;
			}
		}

		Assert::assertTrue(
			$found,
			__METHOD__
			. " Could not find expected resource '$path' in the trash"
		);
	}

	/**
	 * Function to check if an element is in the trashbin
	 *
	 * @param string|null $user
	 * @param string|null $originalPath
	 *
	 * @return bool
	 * @throws Exception
	 */
	private function isInTrash(?string $user, ?string $originalPath): bool {
		$listing = $this->listTrashbinFolder($user);

		// we don't care if the test step writes a leading "/" or not
		$originalPath = \ltrim($originalPath, '/');

		foreach ($listing as $entry) {
			if ($entry['original-location'] !== null && \ltrim($entry['original-location'], '/') === $originalPath) {
				return true;
			}
		}
		return false;
	}

	/**
	 * @param string $user
	 * @param string $trashItemHRef
	 * @param string $destinationPath
	 * @param string|null $asUser - To send request as another user
	 * @param string|null $password
	 *
	 * @return ResponseInterface
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	private function sendUndeleteRequest(
		string $user,
		string $trashItemHRef,
		string $destinationPath,
		?string $asUser = null,
		?string $password = null
	): ResponseInterface {
		$asUser = $asUser ?? $user;
		$password = $password ?? $this->featureContext->getPasswordForUser($asUser);
		$destinationPath = \trim($destinationPath, '/');
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPathVersion = $this->featureContext->getDavPathVersion();

		$suffixPath = $asUser;
		if ($davPathVersion === WebDavHelper::DAV_VERSION_SPACES) {
			if (\str_starts_with($destinationPath, "Shares/")) {
				$suffixPath = $this->featureContext->spacesContext->getSpaceIdByName($user, "Shares");
				$destinationPath = \str_replace("Shares/", "", $destinationPath);
			} else {
				$suffixPath = WebDavHelper::getPersonalSpaceIdForUser(
					$baseUrl,
					$asUser,
					$password,
					$this->featureContext->getStepLineRef()
				);
			}
		}
		$destinationDavPath = WebDavHelper::getDavPath($davPathVersion, $suffixPath);
		$destination = "$baseUrl/$destinationDavPath/$destinationPath";

		$trashItemHRef = \ltrim($this->convertTrashbinHref($trashItemHRef), "/");
		$headers['Destination'] = $destination;
		return $this->featureContext->makeDavRequest(
			$user,
			'MOVE',
			$trashItemHRef,
			$headers,
			null,
			null,
			'trash-bin',
			$davPathVersion,
			false,
			$password,
			[],
			$asUser
		);
	}

	/**
	 * @param string $user
	 * @param string $originalPath
	 * @param string|null $destinationPath
	 * @param string|null $asUser - To send request as another user
	 * @param string|null $password
	 *
	 * @return ResponseInterface
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	private function restoreElement(
		string $user,
		string $originalPath,
		?string $destinationPath = null,
		?string $asUser = null,
		?string $password = null
	): ResponseInterface {
		$asUser = $asUser ?? $user;
		$listing = $this->listTrashbinFolder($user);
		$originalPath = \trim($originalPath, '/');
		if ($destinationPath === null) {
			$destinationPath = $originalPath;
		}
		foreach ($listing as $entry) {
			if ($entry['original-location'] === $originalPath) {
				return $this->sendUndeleteRequest(
					$user,
					$entry['href'],
					$destinationPath,
					$asUser,
					$password
				);
			}
		}
		// The requested element to restore was not even in the trashbin.
		// Throw an exception, because there was not any API call, and so there
		// is also no up-to-date response to examine in later test steps.
		throw new \Exception(
			__METHOD__
			. " cannot restore from trashbin because no element was found for user $user at original path $originalPath"
		);
	}

	/**
	 *
	 * @param $user string
	 * @param $originalPath string
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :user restores the folder/file with original path :originalPath without specifying the destination using the trashbin API')]
	public function userRestoresResourceWithOriginalPathWithoutSpecifyingDestinationUsingTrashbinApi(
		string $user,
		string $originalPath
	): void {
		$listing = $this->listTrashbinFolder($user);
		$originalPath = \trim($originalPath, '/');

		foreach ($listing as $entry) {
			if ($entry['original-location'] === $originalPath) {
				$trashItemHRef = $this->convertTrashbinHref($entry['href']);
				$this->featureContext->setResponse(
					$this->featureContext->makeDavRequest(
						$user,
						'MOVE',
						$trashItemHRef,
						[],
						null,
						null,
						'trash-bin'
					)
				);
			}
		}
	}

	/**
	 *
	 * @param string|null $user
	 * @param string $originalPath
	 *
	 * @return void
	 * @throws JsonException
	 * @throws Exception
	 */
	#[When('/^user "([^"]*)" restores the (?:file|folder|entry) with original path "([^"]*)" using the trashbin API$/')]
	public function elementInTrashIsRestored(?string $user, string $originalPath): void {
		$user = $this->featureContext->getActualUsername($user);
		$this->featureContext->setResponse($this->restoreElement($user, $originalPath));
	}

	/**
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('/^user "([^"]*)" restores the following (?:files|folders|entries) with original path$/')]
	public function userRestoresFollowingFiles(string $user, TableNode $table): void {
		$this->featureContext->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		foreach ($paths as $originalPath) {
			$user = $this->featureContext->getActualUsername($user);
			$this->featureContext->setResponse($this->restoreElement($user, $originalPath["path"]));
			$this->featureContext->pushToLastStatusCodesArrays();
		}
	}

	/**
	 *
	 * @param string $user
	 * @param string $originalPath
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Given('/^user "([^"]*)" has restored the (?:file|folder|entry) with original path "([^"]*)"$/')]
	public function elementInTrashHasBeenRestored(string $user, string $originalPath): void {
		$response = $this->restoreElement($user, $originalPath);
		$this->featureContext->theHTTPStatusCodeShouldBe(201, "", $response);
		if ($this->isInTrash($user, $originalPath)) {
			throw new Exception("File previously located at $originalPath is still in the trashbin");
		}
	}

	/**
	 *
	 * @param string|null $user
	 * @param string|null $originalPath
	 * @param string|null $destinationPath
	 *
	 * @return void
	 * @throws JsonException
	 * @throws Exception
	 */
	#[When('/^user "([^"]*)" restores the (?:file|folder|entry) with original path "([^"]*)" to "([^"]*)" using the trashbin API$/')]
	public function userRestoresTheFileWithOriginalPathToUsingTheTrashbinApi(
		?string $user,
		?string $originalPath,
		?string $destinationPath
	): void {
		$user = $this->featureContext->getActualUsername($user);
		$this->featureContext->setResponse($this->restoreElement($user, $originalPath, $destinationPath));
	}

	/**
	 *
	 * @param string|null $user
	 * @param string|null $originalPath
	 *
	 * @return void
	 * @throws JsonException
	 * @throws Exception
	 */
	#[Then('/^as "([^"]*)" the (?:file|folder|entry) with original path "([^"]*)" should exist in the trashbin$/')]
	public function elementIsInTrashCheckingOriginalPath(
		?string $user,
		?string $originalPath
	): void {
		$user = $this->featureContext->getActualUsername($user);
		Assert::assertTrue(
			$this->isInTrash($user, $originalPath),
			"File previously located at $originalPath wasn't found in the trashbin of user $user"
		);
	}

	/**
	 *
	 * @param string|null $user
	 * @param string $originalPath
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('/^as "([^"]*)" the (?:file|folder|entry) with original path "([^"]*)" should not exist in the trashbin/')]
	public function elementIsNotInTrashCheckingOriginalPath(
		?string $user,
		string $originalPath
	): void {
		$user = $this->featureContext->getActualUsername($user);
		Assert::assertFalse(
			$this->isInTrash($user, $originalPath),
			"File previously located at $originalPath was found in the trashbin of user $user"
		);
	}

	/**
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('/^as "([^"]*)" the (?:files|folders|entries) with following original paths should not exist in the trashbin$/')]
	public function followingElementsAreNotInTrashCheckingOriginalPath(
		string $user,
		TableNode $table
	): void {
		$this->featureContext->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		foreach ($paths as $originalPath) {
			$user = $this->featureContext->getActualUsername($user);
			Assert::assertFalse(
				$this->isInTrash($user, $originalPath["path"]),
				"File previously located at " . $originalPath["path"] . " was found in the trashbin of user $user"
			);
		}
	}

	/**
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('/^as "([^"]*)" the (?:files|folders|entries) with following original paths should exist in the trashbin$/')]
	public function followingElementsAreInTrashCheckingOriginalPath(
		string $user,
		TableNode $table
	): void {
		$this->featureContext->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		foreach ($paths as $originalPath) {
			$user = $this->featureContext->getActualUsername($user);
			Assert::assertTrue(
				$this->isInTrash($user, $originalPath["path"]),
				"File previously located at " . $originalPath["path"] . " wasn't found in the trashbin of user $user"
			);
		}
	}

	/**
	 * Finds the first trashed entry matching the given name
	 *
	 * @param string $user
	 * @param string $name
	 *
	 * @return array|null real entry name with timestamp suffix or null if not found
	 * @throws Exception
	 */
	private function findFirstTrashedEntry(string $user, string $name): ?array {
		$listing = $this->listTrashbinFolder($user);

		foreach ($listing as $entry) {
			if ($entry['name'] === $name) {
				return $entry;
			}
		}

		return null;
	}

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
		// Get all the contexts you need in this context
		$this->featureContext = BehatHelper::getContext($scope, $environment, 'FeatureContext');
	}

	/**
	 *
	 * @param string $resource file or folder in trashbin
	 *
	 * @return void
	 */
	#[Then('/^the deleted (?:file|folder) "([^"]*)" should have the correct deletion mtime in the response$/')]
	public function theDeletedFileFolderShouldHaveCorrectDeletionMtimeInTheResponse(string $resource): void {
		$files = $this->getTrashbinContentFromResponseXml(
			HttpRequestHelper::getResponseXml($this->featureContext->getResponse())
		);

		$found = false;
		$expectedMtime = $this->featureContext->getLastUploadDeleteTime();
		$responseMtime = '';

		foreach ($files as $file) {
			if (\ltrim($resource, "/") === \ltrim((string)$file['original-location'], "/")) {
				$responseMtime = $file['mtime'];
				$mtime_difference = \abs((int)\trim((string)$expectedMtime) - (int)\trim($responseMtime));

				if ($mtime_difference <= 2) {
					$found = true;
					break;
				}
			}
		}
		Assert::assertTrue(
			$found,
			"$resource expected to be listed in response with mtime '$expectedMtime' but found '$responseMtime'"
		);
	}
}
