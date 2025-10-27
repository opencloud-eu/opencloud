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

use Behat\Gherkin\Node\PyStringNode;
use Behat\Gherkin\Node\TableNode;
use GuzzleHttp\Exception\GuzzleException;
use PHPUnit\Framework\Assert;
use Psr\Http\Message\ResponseInterface;
use GuzzleHttp\Stream\StreamInterface;
use TestHelpers\OcHelper;
use TestHelpers\UploadHelper;
use TestHelpers\WebDavHelper;
use TestHelpers\HttpRequestHelper;
use TestHelpers\Asserts\WebDav as WebDavAssert;
use TestHelpers\GraphHelper;

/**
 * WebDav functions
 */
trait WebDav {
	// defaults to the new DAV path
	// maybe we want to make spaces the default in the future
	private int $currentDAVPath = WebDavHelper::DAV_VERSION_NEW;

	/**
	 * @var ResponseInterface[]
	 */
	private array $uploadResponses;

	/**
	 * @var int|string|null
	 */
	private $storedFileID = null;

	private ?int $lastUploadDeleteTime = null;

	/**
	 * add resource created by admin in an array
	 * This array is used while cleaning up the resource created by admin during test run
	 * As of now it tracks only for (files|folder) creation
	 * This can be expanded and modified to track other actions like (upload, deleted.)
	 */
	private array $adminResources = [];

	/**
	 * response content parsed into a SimpleXMLElement
	 */
	private ?SimpleXMLElement $responseXmlObject = null;

	private int $httpRequestTimeout = 0;

	/**
	 * The ability to do requests with depth infinity is disabled by default.
	 * This remembers when the setting dav.propfind.depth_infinity has been
	 * enabled, so that test code can make use of it as appropriate.
	 */
	private bool $davPropfindDepthInfinityEnabled = false;

	private array $filePreviews = [];

	/**
	 * @param string $user
	 *
	 * @return string
	 */
	public function getFilePreviewContent(string $user): string {
		return $this->filePreviews[$user];
	}

	/**
	 * @param string $user
	 * @param string $previewContent
	 *
	 * @return void
	 */
	public function setFilePreviewContent(string $user, string $previewContent): void {
		$this->filePreviews[$user] = $previewContent;
	}

	/**
	 * @return void
	 */
	public function davPropfindDepthInfinityEnabled(): void {
		$this->davPropfindDepthInfinityEnabled = true;
	}

	/**
	 * @return void
	 */
	public function davPropfindDepthInfinityDisabled(): void {
		$this->davPropfindDepthInfinityEnabled = false;
	}

	/**
	 * @return bool
	 */
	public function davPropfindDepthInfinityIsEnabled(): bool {
		return $this->davPropfindDepthInfinityEnabled;
	}

	/**
	 * @param int $lastUploadDeleteTime
	 *
	 * @return void
	 */
	public function setLastUploadDeleteTime(int $lastUploadDeleteTime): void {
		$this->lastUploadDeleteTime = $lastUploadDeleteTime;
	}

	/**
	 * @return number
	 */
	public function getLastUploadDeleteTime(): int {
		return $this->lastUploadDeleteTime;
	}

	/**
	 * @param string $fileID
	 *
	 * @return void
	 */
	public function setStoredFileID(string $fileID): void {
		$this->storedFileID = $fileID;
	}

	/**
	 * @return string
	 */
	public function getStoredFileID(): string {
		return $this->storedFileID;
	}

	/**
	 * @param SimpleXMLElement|null $xmlObject
	 *
	 * @return string the etag or an empty string if the getetag property does not exist
	 */
	public function getEtagFromResponseXmlObject(?SimpleXMLElement $xmlObject = null): string {
		$xmlObject = $xmlObject ?? HttpRequestHelper::getResponseXml($this->getResponse());
		$xmlPart = $xmlObject->xpath("//d:prop/d:getetag");
		if (!\is_array($xmlPart) || (\count($xmlPart) === 0)) {
			return '';
		}
		return $xmlPart[0]->__toString();
	}

	/**
	 *
	 * @param string $eTag
	 *
	 * @return boolean
	 */
	public function isEtagValid(string $eTag): bool {
		if (\preg_match("/^\"[a-f0-9:.]{1,32}\"$/", $eTag)
		) {
			return true;
		} else {
			return false;
		}
	}

	/**
	 * @Given /^using (old|new|spaces) DAV path$/
	 *
	 * @param string $davChoice
	 *
	 * @return void
	 */
	public function usingOldOrNewOrSpacesDavPath(string $davChoice): void {
		switch ($davChoice) {
			case 'old':
				$this->currentDAVPath = WebDavHelper::DAV_VERSION_OLD;
				break;
			case 'new':
				$this->currentDAVPath = WebDavHelper::DAV_VERSION_NEW;
				break;
			case 'spaces':
				$this->currentDAVPath = WebDavHelper::DAV_VERSION_SPACES;
				break;
			default:
				throw new Exception("Invalid DAV path: $davChoice");
		}
	}

	/**
	 * gives the DAV path of a file including the subfolder of the webserver
	 * e.g. when the server runs in `http://localhost/opencloud/`
	 * this function will return `opencloud/webdav/prueba.txt`
	 *
	 * @param string $user
	 * @param string $spaceId
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function getFullDavFilesPath(string $user, ?string $spaceId = null): string {
		if ($spaceId === null && $this->getDavPathVersion() === WebDavHelper::DAV_VERSION_SPACES) {
			$spaceId = WebDavHelper::getPersonalSpaceIdForUser(
				$this->getBaseUrl(),
				$user,
				$this->getPasswordForUser($user),
				$this->getStepLineRef()
			);
		}

		$davPathVersion = $this->getDavPathVersion();
		$suffixPath = $user;
		if ($davPathVersion === WebDavHelper::DAV_VERSION_SPACES) {
			$suffixPath = $spaceId;
		}

		$davPath = WebDavHelper::getDavPath($davPathVersion, $suffixPath);
		$path = "{$this->getBasePath()}/$davPath";
		$path = WebDavHelper::sanitizeUrl($path);
		return \ltrim($path, "/");
	}

	/**
	 * @param string $token
	 *
	 * @return string
	 */
	public function getPublicLinkDavPath(string $token): string {
		$davPath = WebDavHelper::getDavPath($this->getDavPathVersion(), $token, "public-files");
		$path = "{$this->getBasePath()}/$davPath";
		$path = WebDavHelper::sanitizeUrl($path);
		return \ltrim($path, "/");
	}

	/**
	 * Select a suitable DAV path version number.
	 * Some endpoints have only existed since a certain point in time, so for
	 * those make sure to return a DAV path version that works for that endpoint.
	 * Otherwise return the currently selected DAV path version.
	 *
	 * @param string|null $for the category of endpoint that the DAV path will be used for
	 *
	 * @return int DAV path version (1, 2 or 3) selected, or appropriate for the endpoint
	 */
	public function getDavPathVersion(?string $for = null): ?int {
		if (\in_array($for, ['systemtags', 'file_versions'])) {
			// 'systemtags' only exists since new DAV
			// 'file_versions' only exists since new DAV
			return WebDavHelper::DAV_VERSION_NEW;
		}
		return $this->currentDAVPath;
	}

	/**
	 * @param string|null $user
	 * @param string|null $method
	 * @param string|null $path
	 * @param array|null $headers
	 * @param StreamInterface|null $body
	 * @param string|null $spaceId
	 * @param string|null $type
	 * @param string|null $davPathVersion
	 * @param bool $stream Set to true to stream a response rather
	 *                     than download it all up-front.
	 * @param string|null $password
	 * @param array|null $urlParameter
	 * @param string|null $doDavRequestAsUser
	 * @param bool|null $isGivenStep
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function makeDavRequest(
		?string $user,
		?string $method,
		?string $path,
		?array $headers,
		$body = null,
		?string $spaceId = null,
		?string $type = "files",
		?int $davPathVersion = null,
		bool $stream = false,
		?string $password = null,
		?array $urlParameter = [],
		?string $doDavRequestAsUser = null,
		?bool $isGivenStep = false
	): ResponseInterface {
		$user = $this->getActualUsername($user);

		if ($davPathVersion === null) {
			$davPathVersion = $this->getDavPathVersion();
		} else {
			$davPathVersion = (int) $davPathVersion;
		}

		if ($password === null) {
			$password = $this->getPasswordForUser($user);
		}

		return WebDavHelper::makeDavRequest(
			$this->getBaseUrl(),
			$user,
			$password,
			$method,
			$path,
			$headers,
			$spaceId,
			$this->getStepLineRef(),
			$body,
			$davPathVersion,
			$type,
			null,
			"basic",
			$stream,
			$this->httpRequestTimeout,
			null,
			$urlParameter,
			$doDavRequestAsUser,
			$isGivenStep
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $folder
	 * @param bool|null $isGivenStep
	 * @param string|null $password
	 * @param string|null $spaceId
	 *
	 * @return ResponseInterface
	 * @throws JsonException | GuzzleException
	 * @throws GuzzleException | JsonException
	 */
	public function createFolder(
		string $user,
		string $folder,
		?bool $isGivenStep = false,
		?string $password = null,
		?string $spaceId=null
	): ResponseInterface {
		return $this->makeDavRequest(
			$user,
			"MKCOL",
			$folder,
			[],
			null,
			$spaceId,
			"files",
			null,
			false,
			$password,
			[],
			null,
			$isGivenStep
		);
	}

	/**
	 * @param string $user
	 * @param string|null $path
	 * @param string|null $doDavRequestAsUser
	 * @param string|null $width
	 * @param string|null $height
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function downloadPreviews(
		string $user,
		?string $path,
		?string $doDavRequestAsUser,
		?string $width,
		?string $height
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$doDavRequestAsUser = $this->getActualUsername($doDavRequestAsUser);
		$doDavRequestAsUser = $doDavRequestAsUser ?? $user;
		$password = $this->getPasswordForUser($doDavRequestAsUser);
		$urlParameter = [
			'x' => $width,
			'y' => $height,
			'preview' => '1'
		];
		return $this->makeDavRequest(
			$user,
			"GET",
			$path,
			[],
			null,
			null,
			"files",
			null,
			false,
			$password,
			$urlParameter,
			$doDavRequestAsUser
		);
	}

	/**
	 * @Then the number of versions should be :arg1
	 *
	 * @param int $number
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theNumberOfVersionsShouldBe(int $number): void {
		$responseXmlObject = HttpRequestHelper::getResponseXml($this->getResponse(), __METHOD__);
		$xmlPart = $responseXmlObject->xpath("//d:getlastmodified");
		$actualNumber = \count($xmlPart);
		Assert::assertEquals(
			$number,
			$actualNumber,
			"Expected number of versions was '$number', but got '$actualNumber'"
		);
	}

	/**
	 * @Then the number of etag elements in the response should be :number
	 *
	 * @param int $number
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theNumberOfEtagElementInTheResponseShouldBe(int $number): void {
		$responseXmlObject = HttpRequestHelper::getResponseXml($this->getResponse());
		$xmlPart = $responseXmlObject->xpath("//d:getetag");
		$actualNumber = \count($xmlPart);
		Assert::assertEquals(
			$number,
			$actualNumber,
			"Expected number of etag elements was '$number', but got '$actualNumber'"
		);
	}

	/**
	 * @param string $user
	 * @param string $fileDestination
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function destinationHeaderValue(string $user, string $fileDestination): string {
		$fileDestination = \ltrim($fileDestination, "/");
		$spaceId = $this->getPersonalSpaceIdForUser($user);
		// If the destination is a share, we need to get the space ID for the Shares space
		if (\str_starts_with($fileDestination, "Shares/")) {
			$spaceId = $this->spacesContext->getSpaceIdByName($user, "Shares");
			if ($this->getDavPathVersion() === WebDavHelper::DAV_VERSION_SPACES) {
				$fileDestination = \preg_replace("/^Shares\//", "", $fileDestination);
			}
		}

		$davPathVersion = $this->getDavPathVersion();
		$suffixPath = $user;
		if ($davPathVersion === WebDavHelper::DAV_VERSION_SPACES) {
			$suffixPath = $spaceId;
		}

		$davPath = WebDavHelper::getDavPath($davPathVersion, $suffixPath);
		$fullUrl = $this->getBaseUrl() . "/$davPath";
		return \rtrim($fullUrl, '/') . '/' . $fileDestination;
	}

	/**
	 * @Given /^user "([^"]*)" has moved (?:file|folder|entry) "([^"]*)" to "([^"]*)"$/
	 *
	 * @param string|null $user
	 * @param string|null $fileSource
	 * @param string|null $fileDestination
	 *
	 * @return void
	 */
	public function userHasMovedFile(
		?string $user,
		?string $fileSource,
		?string $fileDestination
	): void {
		$response = $this->moveResource($user, $fileSource, $fileDestination);
		$actualStatusCode = $response->getStatusCode();
		$this->theHTTPStatusCodeShouldBe(
			201,
			" Failed moving resource '$fileSource' to '$fileDestination'." .
			" Expected status code was 201 but got '$actualStatusCode' ",
			$response
		);
	}

	/**
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 *
	 * @return ResponseInterface
	 */
	public function moveResource(string $user, string $source, string $destination) {
		$user = $this->getActualUsername($user);
		$headers['Destination'] = $this->destinationHeaderValue(
			$user,
			$destination
		);
		return $this->makeDavRequest(
			$user,
			"MOVE",
			$source,
			$headers
		);
	}

	/**
	 * @When user :user moves file :source to :destination using the WebDAV API
	 * @When user :user moves folder :source to :destination using the WebDAV API
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userMovesFileOrFolderUsingTheWebDavAPI(
		string $user,
		string $source,
		string $destination
	): void {
		$response = $this->moveResource($user, $source, $destination);
		$this->setResponse($response);
		$this->pushToLastHttpStatusCodesArray();
	}

	/**
	 * @When user :user moves the following file using the WebDAV API
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userMovesTheFollowingFileUsingTheWebdavApi(string $user, TableNode $table): void {
		$this->verifyTableNodeColumns($table, ["source",  "destination"]);
		$rows = $table->getHash();
		foreach ($rows as $row) {
			$response = $this->moveResource($user, $row["source"], $row["destination"]);
			$this->setResponse($response);
			$this->pushToLastHttpStatusCodesArray(
				(string) $response->getStatusCode()
			);
		}
	}

	/**
	 * @When /^user "([^"]*)" moves the following (?:files|folders|entries)\s?(asynchronously|) using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $type "asynchronously" or empty
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userMovesFollowingFileUsingTheAPI(
		string $user,
		string $type,
		TableNode $table
	): void {
		$this->verifyTableNodeColumns($table, ["from", "to"]);
		$paths = $table->getHash();

		foreach ($paths as $file) {
			$response = $this->moveResource($user, $file['from'], $file['to']);
			$this->pushToLastHttpStatusCodesArray(
				(string) $response->getStatusCode()
			);
		}
	}

	/**
	 * @Then /^user "([^"]*)" should be able to rename (file|folder|entry) "([^"]*)" to "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $entry
	 * @param string $source
	 * @param string $destination
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theUserShouldBeAbleToRenameEntryTo(
		string $user,
		string $entry,
		string $source,
		string $destination
	): void {
		$user = $this->getActualUsername($user);
		$this->checkFileOrFolderExistsForUser($user, $entry, $source);
		$response = $this->moveResource($user, $source, $destination);
		$this->theHTTPStatusCodeShouldBeBetween(201, 204, $response);
		$this->checkFileOrFolderDoesNotExistsForUser($user, $entry, $source);
		$this->checkFileOrFolderExistsForUser($user, $entry, $destination);
	}

	/**
	 * @Then /^user "([^"]*)" should not be able to rename (file|folder|entry) "([^"]*)" to "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $entry
	 * @param string $source
	 * @param string $destination
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theUserShouldNotBeAbleToRenameEntryTo(
		string $user,
		string $entry,
		string $source,
		string $destination
	): void {
		$this->checkFileOrFolderExistsForUser($user, $entry, $source);
		$response = $this->moveResource($user, $source, $destination);
		$this->theHTTPStatusCodeShouldBeBetween(400, 499, $response);
		$this->checkFileOrFolderExistsForUser($user, $entry, $source);
		$this->checkFileOrFolderDoesNotExistsForUser($user, $entry, $destination);
	}

	/**
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fileDestination
	 *
	 * @return ResponseInterface
	 */
	public function copyFile(
		string $user,
		string $fileSource,
		string $fileDestination
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$headers['Destination'] = $this->destinationHeaderValue(
			$user,
			$fileDestination
		);
		return $this->makeDavRequest(
			$user,
			"COPY",
			$fileSource,
			$headers
		);
	}

	/**
	 * @When /^user "([^"]*)" copies (?:file|folder) "([^"]*)" to "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fileDestination
	 *
	 * @return void
	 */
	public function userCopiesFileUsingTheAPI(
		string $user,
		string $fileSource,
		string $fileDestination
	): void {
		$response = $this->copyFile($user, $fileSource, $fileDestination);
		$this->setResponse($response);
		$this->pushToLastHttpStatusCodesArray();
	}

	/**
	 * @Given /^user "([^"]*)" has copied (?:file|folder) "([^"]*)" to "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fileDestination
	 *
	 * @return void
	 */
	public function userHasCopiedFileUsingTheAPI(
		string $user,
		string $fileSource,
		string $fileDestination
	): void {
		$response = $this->copyFile($user, $fileSource, $fileDestination);
		$this->theHTTPStatusCodeShouldBe(
			["201", "204"],
			"HTTP status code was not 201 or 204 while trying to copy resource "
			. "'$fileSource' to '$fileDestination' for user '$user'",
			$response
		);
	}

	/**
	 * @param string $user
	 * @param string $fileSource
	 * @param string $range
	 *
	 * @return ResponseInterface
	 */
	public function downloadFileWithRange(string $user, string $fileSource, string $range): ResponseInterface {
		$user = $this->getActualUsername($user);
		$headers['Range'] = $range;
		return $this->makeDavRequest(
			$user,
			"GET",
			$fileSource,
			$headers
		);
	}

	/**
	 * @When /^user "([^"]*)" downloads file "([^"]*)" with range "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $range
	 *
	 * @return void
	 */
	public function userDownloadsFileWithRangeUsingWebDavApi(string $user, string $fileSource, string $range): void {
		$this->setResponse($this->downloadFileWithRange($user, $fileSource, $range));
	}

	/**
	 * @When the user waits for :time seconds for postprocessing to finish
	 *
	 * @param int $time
	 *
	 * @return void
	 */
	public function waitForCertainSeconds(int $time): void {
		\sleep($time);
	}

	/**
	 * @Then /^user "([^"]*)" using password "([^"]*)" should not be able to download file "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $password
	 * @param string $fileName
	 *
	 * @return void
	 */
	public function userUsingPasswordShouldNotBeAbleToDownloadFile(
		string $user,
		string $password,
		string $fileName
	): void {
		$user = $this->getActualUsername($user);
		$password = $this->getActualPassword($password);
		$response = $this->downloadFileAsUserUsingPassword($user, $fileName, $password);
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

	/**
	 * @Then /^user "([^"]*)" should not be able to download file "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $fileName
	 *
	 * @return void
	 * @throws JsonException
	 */
	public function userShouldNotBeAbleToDownloadFile(
		string $user,
		string $fileName
	): void {
		$user = $this->getActualUsername($user);
		$password = $this->getPasswordForUser($user);
		$response = $this->downloadFileAsUserUsingPassword($user, $fileName, $password);
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

	/**
	 * @Then the size of the downloaded file should be :size bytes
	 *
	 * @param string $size
	 *
	 * @return void
	 */
	public function sizeOfDownloadedFileShouldBe(string $size): void {
		$actualSize = \strlen((string) $this->response->getBody());
		Assert::assertEquals(
			$size,
			$actualSize,
			"Expected size of the downloaded file was '$size' but got '$actualSize'"
		);
	}

	/**
	 * @Then /^the downloaded content should end with "([^"]*)"$/
	 *
	 * @param string $content
	 *
	 * @return void
	 */
	public function downloadedContentShouldEndWith(string $content): void {
		$actualContent = \substr((string) $this->response->getBody(), -\strlen($content));
		Assert::assertEquals(
			$content,
			$actualContent,
			"The downloaded content was expected to end with '$content', but actually ended with '$actualContent'."
		);
	}

	/**
	 * @Then /^the downloaded content should be "([^"]*)"$/
	 *
	 * @param string $content
	 *
	 * @return void
	 */
	public function downloadedContentShouldBe(string $content): void {
		$this->checkDownloadedContentMatches($content, '', $this->getResponse());
	}

	/**
	 * @param string $expectedContent
	 * @param string $extraErrorText
	 * @param ResponseInterface|null $response
	 *
	 * @return void
	 */
	public function checkDownloadedContentMatches(
		string $expectedContent,
		string $extraErrorText = "",
		?ResponseInterface $response = null
	): void {
		$response = $response ?? $this->response;
		$actualContent = (string) $response->getBody();
		// For this test we really care about the content.
		// A separate "Then" step can specifically check the HTTP status.
		// But if the content is wrong (e.g. empty) then it is useful to
		// report the HTTP status to give some clue what might be the problem.
		$actualStatus = $response->getStatusCode();
		if ($extraErrorText !== "") {
			$extraErrorText .= "\n";
		}
		Assert::assertEquals(
			$expectedContent,
			$actualContent,
			$extraErrorText . "The content was expected to be '$expectedContent', but actually is "
			. "'$actualContent'. HTTP status was $actualStatus"
		);
	}

	/**
	 * @Then the content in the response should match the following content:
	 *
	 * @param PyStringNode $content
	 *
	 * @return void
	 */
	public function theContentInTheResponseShouldMatchTheFollowingContent(PyStringNode $content): void {
		$this->checkDownloadedContentMatches($content->getRaw(), '', $this->getResponse());
	}

	/**
	 * @Then the content in the response should include the following content:
	 *
	 * @param PyStringNode $content
	 *
	 * @return void
	 */
	public function theContentInTheResponseShouldIncludeTheFollowingContent(PyStringNode $content): void {
		Assert::assertStringContainsString(
			$content->getRaw(),
			(string) $this->response->getBody()
		);
	}

	/**
	 * @Then /^if the HTTP status code was "([^"]*)" then the downloaded content for multipart byterange should be:$/
	 *
	 * @param int $statusCode
	 * @param PyStringNode $content
	 *
	 * @return void
	 *
	 */
	public function theDownloadedContentForMultipartByteRangeShouldBe(int $statusCode, PyStringNode $content): void {
		$actualStatusCode = $this->response->getStatusCode();
		if ($actualStatusCode === $statusCode) {
			$actualContent = (string) $this->response->getBody();
			$pattern = ["/--\w*/", "/\s*/m"];
			$actualContent = \preg_replace($pattern, "", $actualContent);
			$content = \preg_replace("/\s*/m", '', $content->getRaw());
			Assert::assertEquals(
				$content,
				$actualContent,
				"The downloaded content was expected to be '$content', but actually is "
				. "'$actualContent'. HTTP status was $actualStatusCode"
			);
		}
	}

	/**
	 * @Then /^if the HTTP status code was "([^"]*)" then the downloaded content should be "([^"]*)"$/
	 *
	 * @param int $statusCode
	 * @param string $content
	 *
	 * @return void
	 */
	public function checkStatusCodeForDownloadedContentShouldBe(int $statusCode, string $content): void {
		$actualStatusCode = $this->response->getStatusCode();
		if ($actualStatusCode === $statusCode) {
			$this->checkDownloadedContentMatches($content, '', $this->getResponse());
		}
	}

	/**
	 * @Then the content of file :fileName for user :user should be :content
	 *
	 * @param string $fileName
	 * @param string $user
	 * @param string $content
	 *
	 * @return void
	 */
	public function contentOfFileForUserShouldBe(string $fileName, string $user, string $content): void {
		$response = $this->downloadFileAsUserUsingPassword($this->getActualUsername($user), $fileName);
		$actualStatus = $response->getStatusCode();
		if ($actualStatus !== 200) {
			throw new Exception(
				"Expected status code to be '200', but got '$actualStatus'"
			);
		}
		$this->checkDownloadedContentMatches($content, '', $response);
	}

	/**
	 * @Then /^the content of the following files for user "([^"]*)" should be "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $content
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function contentOfFollowingFilesShouldBe(string $user, string $content, TableNode $table): void {
		$this->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		$user = $this->getActualUsername($user);
		foreach ($paths as $file) {
			$response = $this->downloadFileAsUserUsingPassword($user, $file["path"]);
			$actualStatus = $response->getStatusCode();
			Assert::assertEquals(
				200,
				$actualStatus,
				"Expected status code to be '200', but got '$actualStatus'"
			);
			$this->checkDownloadedContentMatches($content, '', $response);
		}
	}

	/**
	 * @Then /^the content of file "([^"]*)" for user "([^"]*)" using password "([^"]*)" should be "([^"]*)"$/
	 *
	 * @param string $fileName
	 * @param string $user
	 * @param string|null $password
	 * @param string $content
	 *
	 * @return void
	 */
	public function contentOfFileForUserUsingPasswordShouldBe(
		string $fileName,
		string $user,
		?string $password,
		string $content
	): void {
		$user = $this->getActualUsername($user);
		$password = $this->getActualPassword($password);
		$response = $this->downloadFileAsUserUsingPassword($user, $fileName, $password);
		$this->checkDownloadedContentMatches($content, '', $response);
	}

	/**
	 * @Then /^the content of file "([^"]*)" for user "([^"]*)" should be:$/
	 *
	 * @param string $fileName
	 * @param string $user
	 * @param PyStringNode $content
	 *
	 * @return void
	 */
	public function contentOfFileForUserShouldBePyString(
		string $fileName,
		string $user,
		PyStringNode $content
	): void {
		$user = $this->getActualUsername($user);
		$response = $this->downloadFileAsUserUsingPassword($user, $fileName);
		$actualStatus = $response->getStatusCode();
		Assert::assertEquals(
			200,
			$actualStatus,
			"Expected status code to be '200', but got '$actualStatus'"
		);
		$this->checkDownloadedContentMatches($content->getRaw(), '', $response);
	}

	/**
	 * @When user :user downloads file :fileName using the WebDAV API
	 * @When user :user tries to download file :fileName using the WebDAV API
	 *
	 * @param string $user
	 * @param string $fileName
	 *
	 * @return void
	 */
	public function userDownloadsFileUsingTheAPI(
		string $user,
		string $fileName
	): void {
		$this->setResponse($this->downloadFileAsUserUsingPassword($user, $fileName));
	}

	/**
	 * @param string $user
	 * @param string $fileName
	 * @param string|null $password
	 * @param array|null $headers
	 * @param string|null $spaceId
	 *
	 * @return ResponseInterface
	 */
	public function downloadFileAsUserUsingPassword(
		string $user,
		string $fileName,
		?string $password = null,
		?array $headers = [],
		?string $spaceId = null,
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$password = $this->getActualPassword($password);
		return $this->makeDavRequest(
			$user,
			'GET',
			$fileName,
			$headers,
			null,
			$spaceId,
			"files",
			null,
			false,
			$password
		);
	}

	/**
	 * @When the public gets the size of the last shared public link using the WebDAV API
	 *
	 * @return void
	 * @throws Exception
	 */
	public function publicGetsSizeOfLastSharedPublicLinkUsingTheWebdavApi(): void {
		$token = ($this->isUsingSharingNG())
		? $this->shareNgGetLastCreatedLinkShareToken() : $this->getLastCreatedPublicShareToken();
		$davPath = WebDavHelper::getDavPath($this->getDavPathVersion(), $token, "public-files");
		$url = "{$this->getBaseUrl()}/$davPath";
		$this->response = HttpRequestHelper::sendRequest(
			$url,
			$this->getStepLineRef(),
			"PROPFIND"
		);
	}

	/**
	 * @When user :user gets the size of file :resource using the WebDAV API
	 *
	 * @param string $user
	 * @param string $resource
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userGetsSizeOfFileUsingTheWebdavApi(string $user, string $resource): void {
		$user = $this->getActualUsername($user);
		$password = $this->getPasswordForUser($user);
		$this->response = WebDavHelper::propfind(
			$this->getBaseUrl(),
			$user,
			$password,
			$resource,
			[],
			$this->getStepLineRef(),
			"0",
			null,
			"files",
			$this->getDavPathVersion()
		);
	}

	/**
	 * @Then the size of the file should be :size
	 *
	 * @param string $size
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theSizeOfTheFileShouldBe(string $size): void {
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$this->response,
			__METHOD__
		);
		$xmlPart = $responseXmlObject->xpath("//d:prop/d:getcontentlength");
		$actualSize = (string) $xmlPart[0];
		Assert::assertEquals(
			$size,
			$actualSize,
			__METHOD__
			. " Expected size of the file was '$size', but got '$actualSize' instead."
		);
	}

	/**
	 * @Then /^the content of file "([^"]*)" for user "([^"]*)" should be "([^"]*)" plus end-of-line$/
	 *
	 * @param string $fileName
	 * @param string $user
	 * @param string $content
	 *
	 * @return void
	 */
	public function contentOfFileForUserShouldBePlusEndOfLine(string $fileName, string $user, string $content): void {
		$user = $this->getActualUsername($user);
		$response = $this->downloadFileAsUserUsingPassword($user, $fileName);
		$actualStatus = $response->getStatusCode();
		Assert::assertEquals(
			200,
			$actualStatus,
			"Expected status code to be '200', but got '$actualStatus'"
		);
		$this->checkDownloadedContentMatches("$content\n", '', $response);
	}

	/**
	 * @Then the following headers should be set
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theFollowingHeadersShouldBeSet(TableNode $table): void {
		$this->verifyTableNodeColumns(
			$table,
			['header', 'value']
		);
		foreach ($table->getColumnsHash() as $header) {
			$headerName = $header['header'];
			$expectedHeaderValue = $header['value'];
			$returnedHeader = $this->response->getHeader($headerName);
			$expectedHeaderValue = $this->substituteInLineCodes($expectedHeaderValue);

			if (empty($returnedHeader)) {
				throw new Exception(
					\sprintf(
						"Missing expected header '%s'",
						$headerName
					)
				);
			}
			$headerValue = $returnedHeader[0];

			Assert::assertEquals(
				$expectedHeaderValue,
				$headerValue,
				__METHOD__
				. " Expected value for header '$headerName' was '$expectedHeaderValue', but got '$headerValue' instead."
			);
		}
	}

	/**
	 * @Then as :user :entry :path should not exist
	 *
	 * @param string $user
	 * @param string $entry
	 * @param string $path
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function asFileOrFolderShouldNotExist(
		string $user,
		string $entry,
		string $path
	): void {
		$this->checkFileOrFolderDoesNotExistsForUser($user, $entry, $path);
	}

	/**
	 * @param string $user
	 * @param string $entry
	 * @param string|null $path
	 * @param string $type
	 *
	 * @return void
	 */
	public function checkFileOrFolderDoesNotExistsForUser(
		string $user,
		string $entry = "file",
		?string $path = null,
		string $type = "files"
	): void {
		$user = $this->getActualUsername($user);
		$path = $this->substituteInLineCodes($path);
		$response = $this->listFolder(
			$user,
			$path,
			'0',
			null,
			null,
			$type
		);
		$statusCode = $response->getStatusCode();
		if ($statusCode < 400 || $statusCode > 499) {
			try {
				$responseXmlObject = HttpRequestHelper::getResponseXml(
					$response,
					__METHOD__
				);
			} catch (Exception $e) {
				Assert::fail(
					"$entry '$path' should not exist. But API returned $statusCode without XML in the body"
				);
			}
			Assert::assertTrue(
				$this->isEtagValid($this->getEtagFromResponseXmlObject($responseXmlObject)),
				"$entry '$path' should not exist. But API returned $statusCode without an etag in the body"
			);
			$isCollection = $responseXmlObject->xpath("//d:prop/d:resourcetype/d:collection");
			if (\count($isCollection) === 0) {
				$actualResourceType = "file";
			} else {
				$actualResourceType = "folder";
			}

			if ($entry === $actualResourceType) {
				Assert::fail(
					"$entry '$path' should not exist. But it does."
				);
			}
		}
	}

	/**
	 * @Then /^as "([^"]*)" the following (files|folders) should not exist$/
	 *
	 * @param string $user
	 * @param string $entry
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function followingFilesShouldNotExist(
		string $user,
		string $entry,
		TableNode $table
	): void {
		$this->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();
		$entry = \rtrim($entry, "s");

		foreach ($paths as $file) {
			$this->checkFileOrFolderDoesNotExistsForUser($user, $entry, $file["path"]);
		}
	}

	/**
	 * @Then as :user :entry :path should exist
	 *
	 * @param string $user
	 * @param string $entry
	 * @param string $path
	 *
	 * @return void
	 * @throws Exception
	 */
	public function asFileOrFolderShouldExist(
		string $user,
		string $entry,
		string $path
	): void {
		$this->checkFileOrFolderExistsForUser($user, $entry, $path);
	}

	/**
	 * @param string $user
	 * @param string $entry
	 * @param string $path
	 * @param string|null $type
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkFileOrFolderExistsForUser(
		string $user,
		string $entry,
		string $path,
		?string $type = "files"
	): void {
		$user = $this->getActualUsername($user);
		$path = $this->substituteInLineCodes($path);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$this->listFolder(
				$user,
				$path,
				'0',
				null,
				null,
				$type
			)
		);

		Assert::assertTrue(
			$this->isEtagValid($this->getEtagFromResponseXmlObject($responseXmlObject)),
			"$entry '$path' expected to exist for user $user but not found"
		);
		$isCollection = $responseXmlObject->xpath("//d:prop/d:resourcetype/d:collection");
		if ($entry === "folder") {
			Assert::assertEquals(\count($isCollection), 1, "Unexpectedly, `$path` is not a folder");
		} elseif ($entry === "file") {
			Assert::assertEquals(\count($isCollection), 0, "Unexpectedly, `$path` is not a file");
		}
	}

	/**
	 * @Then /^as "([^"]*)" the following (files|folders) should exist$/
	 *
	 * @param string $user
	 * @param string $entry
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function followingFilesOrFoldersShouldExist(
		string $user,
		string $entry,
		TableNode $table
	): void {
		$this->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();
		$entry = \rtrim($entry, "s");

		foreach ($paths as $file) {
			$this->checkFileOrFolderExistsForUser($user, $entry, $file["path"]);
		}
	}

	/**
	 *
	 * @param string $user
	 * @param string $entry
	 * @param string $path
	 * @param string $type
	 *
	 * @return bool
	 */
	public function fileOrFolderExists(
		string $user,
		string $entry,
		string $path,
		string $type = "files"
	): bool {
		try {
			$this->checkFileOrFolderExistsForUser($user, $entry, $path, $type);
			return true;
		} catch (Exception $e) {
			return false;
		}
	}

	/**
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $folderDepth requires 1 to see elements without children
	 * @param array|null $properties
	 * @param string|null $spaceId
	 * @param string $type
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function listFolder(
		string $user,
		string $path,
		string $folderDepth,
		?array $properties = null,
		?string $spaceId = null,
		string $type = "files"
	): ResponseInterface {
		return WebDavHelper::listFolder(
			$this->getBaseUrl(),
			$this->getActualUsername($user),
			$this->getPasswordForUser($user),
			$path,
			$folderDepth,
			$spaceId,
			$this->getStepLineRef(),
			$properties,
			$type,
			$this->getDavPathVersion()
		);
	}

	/**
	 * @Then /^user "([^"]*)" should (not|)\s?see the following elements$/
	 *
	 * @param string $user
	 * @param string $shouldOrNot
	 * @param TableNode $elements
	 *
	 * @return void
	 * @throws InvalidArgumentException|Exception
	 *
	 */
	public function userShouldSeeTheElements(string $user, string $shouldOrNot, TableNode $elements): void {
		$should = ($shouldOrNot !== "not");
		$this->checkElementList($user, $elements, $should);
	}

	/**
	 * asserts that the user can or cannot see a list of files/folders by propfind
	 *
	 * @param string $user
	 * @param TableNode $elements
	 * @param boolean $expectedToBeListed
	 *
	 * @return void
	 * @throws InvalidArgumentException
	 * @throws Exception
	 *
	 */
	public function checkElementList(
		string $user,
		TableNode $elements,
		bool $expectedToBeListed = true
	): void {
		$user = $this->getActualUsername($user);
		$this->verifyTableNodeColumnsCount($elements, 1);
		$elementRows = $elements->getRows();
		$elementsSimplified = $this->simplifyArray($elementRows);
		if ($this->davPropfindDepthInfinityIsEnabled()) {
			// get a full "infinite" list of the user's root folder in one request
			// and process that to check the elements (resources)
			$responseXmlObject = HttpRequestHelper::getResponseXml(
				$this->listFolder(
					$user,
					"/",
					"infinity"
				)
			);
			foreach ($elementsSimplified as $expectedElement) {
				// Allow the table of expected elements to have entries that do
				// not have to specify the "implied" leading slash, or have multiple
				// leading slashes, to make scenario outlines more flexible
				$expectedElement = $this->encodePath($expectedElement);
				$expectedElement = "/" . \ltrim($expectedElement, "/");
				$webdavPath = "/" . $this->getFullDavFilesPath($user) . $expectedElement;
				$element = $responseXmlObject->xpath(
					"//d:response/d:href[text() = \"$webdavPath\"]"
				);
				if ($expectedToBeListed
					&& (!isset($element[0]) || urldecode($element[0]->__toString()) !== urldecode($webdavPath))
				) {
					Assert::fail(
						"$webdavPath is not in propfind answer but should be"
					);
				} elseif (!$expectedToBeListed && isset($element[0])
				) {
					Assert::fail(
						"$webdavPath is in propfind answer but should not be"
					);
				}
			}
		} else {
			// do a PROPFIND for each element
			foreach ($elementsSimplified as $elementToRequest) {
				// Allow the table of expected elements to have entries that do
				// not have to specify the "implied" leading slash, or have multiple
				// leading slashes, to make scenario outlines more flexible
				$elementToRequest = "/" . \ltrim($elementToRequest, "/");
				// Note: in the request we ask to do a PROPFIND on a resource like:
				//       /some-folder with spaces/sub-folder
				// but the response has encoded values for the special characters like:
				//       /some-folder%20with%20spaces/sub-folder
				// So we need both $elementToRequest and $expectedElement
				$expectedElement = $this->encodePath($elementToRequest);
				$responseXmlObject = HttpRequestHelper::getResponseXml(
					$this->listFolder(
						$user,
						$elementToRequest,
						'1'
					)
				);

				$webdavPath = "/" . $this->getFullDavFilesPath($user) . $expectedElement;
				// return xmlobject that matches the x-path pattern.
				$element = $responseXmlObject->xpath(
					"//d:response/d:href"
				);
				// check the first element because the requested resource will always be the first one
				if (!$expectedToBeListed && isset($element[0])
				) {
					Assert::fail(
						"$webdavPath is in propfind answer but should not be"
					);
				};
				if ($expectedToBeListed) {
					$elementSanitized =  rtrim($element[0]->__toString(), '/');
					if (!isset($element[0]) || urldecode($elementSanitized) !== urldecode($webdavPath)) {
						Assert::fail(
							"$webdavPath is not in propfind answer but should be"
						);
					}
				}
			}
		}
	}

	/**
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $spaceId
	 * @param bool|null $isGivenStep
	 *
	 * @return ResponseInterface
	 */
	public function uploadFile(
		string $user,
		string $source,
		string $destination,
		string $spaceId = null,
		?bool $isGivenStep = false
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$file = \fopen($this->acceptanceTestsDirLocation() . $source, 'r');
		$this->pauseUploadDelete();
		$response = $this->makeDavRequest(
			$user,
			"PUT",
			$destination,
			[],
			$file,
			$spaceId,
			"files",
			null,
			false,
			null,
			[],
			null,
			$isGivenStep
		);
		$this->lastUploadDeleteTime = \time();
		return $response;
	}

	/**
	 * @When user :user uploads file :source to :destination using the WebDAV API
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 *
	 * @return void
	 */
	public function userUploadsAFileToUsingWebDavApi(
		string $user,
		string $source,
		string $destination
	): void {
		$response = $this->uploadFile($user, $source, $destination);
		$this->setResponse($response);
		$this->pushToLastHttpStatusCodesArray(
			(string) $response->getStatusCode()
		);
	}

	/**
	 * @Given user :user has uploaded file :source to :destination
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 *
	 * @return array
	 */
	public function userHasUploadedAFileTo(string $user, string $source, string $destination): array {
		$response = $this->uploadFile($user, $source, $destination, null, true);
		$this->theHTTPStatusCodeShouldBe(
			["201", "204"],
			"HTTP status code was not 201 or 204 while trying to upload file "
			. "'$source' to '$destination' for user '$user'",
			$response
		);
		return $response->getHeader('oc-fileid');
	}

	/**
	 * Upload file as a user with different headers
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param array|null $headers
	 * @param int|null $noOfChunks Only use for chunked upload when $this->chunkingToUse is not null
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function uploadFileWithHeaders(
		string $user,
		string $source,
		string $destination,
		?array $headers = [],
		?int $noOfChunks = 0
	): ResponseInterface {
		$doChunkUpload = true;
		if ($noOfChunks <= 0) {
			$doChunkUpload = false;
		}

		$this->pauseUploadDelete();
		$response = UploadHelper::upload(
			$this->getBaseUrl(),
			$this->getActualUsername($user),
			$this->getUserPassword($user),
			$source,
			$destination,
			$this->getStepLineRef(),
			$headers,
			$this->getDavPathVersion(),
			$doChunkUpload,
			$noOfChunks,
		);
		$this->lastUploadDeleteTime = \time();
		return $response;
	}

	/**
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param integer $noOfChunks
	 * @param boolean $async
	 * @param array|null $headers
	 *
	 * @return ResponseInterface
	 */
	public function userUploadsAFileInChunk(
		string $user,
		string $source,
		string $destination,
		int $noOfChunks = 2,
		bool $async = false,
		?array $headers = []
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		Assert::assertGreaterThan(
			0,
			$noOfChunks,
			"What does it mean to have $noOfChunks chunks?"
		);

		if ($async === true) {
			$headers['OC-LazyOps'] = 'true';
		}
		return $this->uploadFileWithHeaders(
			$user,
			$this->acceptanceTestsDirLocation() . $source,
			$destination,
			$headers,
			$noOfChunks
		);
	}

	/**
	 * @When /^user "([^"]*)" uploads file "([^"]*)" to "([^"]*)" in (\d+) chunks using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param int $noOfChunks
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userUploadsAFileToWithChunks(
		string $user,
		string $source,
		string $destination,
		int $noOfChunks = 2
	): void {
		$response = $this->userUploadsAFileInChunk($user, $source, $destination, $noOfChunks);
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @Then the HTTP status code of responses on all endpoints should be :statusCode
	 *
	 * @param int $statusCode
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theHTTPStatusCodeOfResponsesOnAllEndpointsShouldBe(int $statusCode): void {
		$duplicateRemovedStatusCodes = \array_unique($this->lastHttpStatusCodesArray);
		if (\count($duplicateRemovedStatusCodes) === 1) {
			Assert::assertSame(
				$statusCode,
				\intval($duplicateRemovedStatusCodes[0]),
				'Responses did not return expected http status code'
			);
			$this->emptyLastHTTPStatusCodesArray();
		} else {
			throw new Exception(
				'Expected same but found different http status codes of last requested responses.' .
				'Found status codes: ' . \implode(',', $this->lastHttpStatusCodesArray)
			);
		}
	}

	/**
	 * @param string $statusCodes a comma-separated string of expected HTTP status codes
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkTheHTTPStatusCodeOfResponsesOnEachEndpoint(string $statusCodes): void {
		$expectedStatusCodes = \explode(',', $statusCodes);
		$actualStatusCodes = $this->lastHttpStatusCodesArray;
		$count = \count($expectedStatusCodes);
		$statusCodesAreAllOk = true;
		if ($count === \count($actualStatusCodes)) {
			for ($i = 0; $i < $count; $i++) {
				$expectedCode = (int)\trim($expectedStatusCodes[$i]);
				$actualCode = (int)$actualStatusCodes[$i];
				if ($expectedCode !== $actualCode) {
					$statusCodesAreAllOk = false;
				}
			}
		} else {
			$statusCodesAreAllOk = false;
		}
		$this->emptyLastHTTPStatusCodesArray();
		Assert::assertTrue(
			$statusCodesAreAllOk,
			'Expected HTTP status codes: "' . $statusCodes .
			'". Found HTTP status codes: "' . \implode(',', $actualStatusCodes) . '"'
		);
	}

	/**
	 * @Then the HTTP status code of responses on each endpoint should be :statusCodes respectively
	 *
	 * @param string $statusCodes a comma-separated string of expected HTTP status codes
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theHTTPStatusCodeOfResponsesOnEachEndpointShouldBe(string $statusCodes): void {
		$this->checkTheHTTPStatusCodeOfResponsesOnEachEndpoint($statusCodes);
	}

	/**
	 * @Then the HTTP status code of responses on each endpoint should be :ocStatusCodes on OpenCloud or :revaStatusCodes on reva
	 *
	 * @param string $ocStatusCodes a comma-separated string of expected HTTP status codes when running on OpenCloud
	 * @param string $revaStatusCodes a comma-separated string of expected HTTP status codes when running on reva
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theHTTPStatusCodeOfResponsesOnEachEndpointShouldBeOcReva(
		string $ocStatusCodes,
		string $revaStatusCodes
	): void {
		if (OcHelper::isTestingOnReva()) {
			$expectedStatusCodes = $revaStatusCodes;
		} else {
			$expectedStatusCodes = $ocStatusCodes;
		}
		$this->checkTheHTTPStatusCodeOfResponsesOnEachEndpoint($expectedStatusCodes);
	}

	/**
	 * @Then the OCS status code of responses on each endpoint should be :statusCode respectively
	 *
	 * @param string $statusCodes
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theOCStatusCodeOfResponsesOnEachEndpointShouldBe(string $statusCodes): void {
		$statusCodes = \explode(',', $statusCodes);
		$count = \count($statusCodes);
		if ($count === \count($this->lastOCSStatusCodesArray)) {
			for ($i = 0; $i < $count; $i++) {
				Assert::assertSame(
					(int)\trim($statusCodes[$i]),
					(int)$this->lastOCSStatusCodesArray[$i],
					'Responses did not return expected OCS status code'
				);
			}
		} else {
			throw new Exception(
				'Expected OCS status codes: "' . \implode(',', $statusCodes) .
				'". Found OCS status codes: "' . \implode(',', $this->lastOCSStatusCodesArray) . '"'
			);
		}
	}

	/**
	 * @Then the OCS status code of responses on all endpoints should be :statusCode
	 *
	 * @param string $statusCode
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theOCSStatusCodeOfResponsesOnAllEndpointsShouldBe(string $statusCode): void {
		$duplicateRemovedStatusCodes = \array_unique($this->lastOCSStatusCodesArray);
		if (\count($duplicateRemovedStatusCodes) === 1) {
			Assert::assertSame(
				\intval($statusCode),
				\intval($duplicateRemovedStatusCodes[0]),
				'Responses did not return expected ocs status code'
			);
			$this->emptyLastOCSStatusCodesArray();
		} else {
			throw new Exception(
				'Expected same but found different ocs status codes of last requested responses.' .
				'Found status codes: ' . \implode(',', $this->lastOCSStatusCodesArray)
			);
		}
	}

	/**
	 * @Then user :user should be able to upload file :source to :destination
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldBeAbleToUploadFileTo(string $user, string $source, string $destination): void {
		$user = $this->getActualUsername($user);
		$response = $this->uploadFile($user, $source, $destination);
		$this->theHTTPStatusCodeShouldBe(
			["201", "204"],
			"HTTP status code was not 201 or 204 while trying to upload file '$destination'",
			$response
		);
		$this->checkFileOrFolderExistsForUser($user, "file", $destination);
	}

	/**
	 * @Then user :user should not be able to upload file :source to :destination
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theUserShouldNotBeAbleToUploadFileTo(string $user, string $source, string $destination): void {
		$fileAlreadyExists = $this->fileOrFolderExists($user, "file", $destination);
		if ($fileAlreadyExists) {
			$response = $this->downloadFileAsUserUsingPassword($user, $destination);
			$initialContent = (string) $response->getBody();
		}
		$response = $this->uploadFile($user, $source, $destination);
		$this->theHTTPStatusCodeShouldBe(["403", "423"], "", $response);
		if ($fileAlreadyExists) {
			$response = $this->downloadFileAsUserUsingPassword($user, $destination);
			$currentContent = (string) $response->getBody();
			Assert::assertSame(
				$initialContent,
				$currentContent,
				__METHOD__
				. " user $user was unexpectedly able to upload $source to $destination - the content has changed:"
			);
		} else {
			$this->checkFileOrFolderDoesNotExistsForUser($user, "file", $destination);
		}
	}

	/**
	 * @param string $user
	 * @param string $destination
	 * @param string $shouldOrNot
	 * @param string|null $exceptChunkingType
	 *
	 * @return void
	 */
	public function checkIfFilesExist(
		string $user,
		string $destination,
		string $shouldOrNot,
		?string $exceptChunkingType = ''
	): void {
		switch ($exceptChunkingType) {
			case 'old':
				$exceptChunkingSuffix = 'olddav-oldchunking';
				break;
			case 'new':
				$exceptChunkingSuffix = 'newdav-newchunking';
				break;
			default:
				$exceptChunkingSuffix = '';
				break;
		}

		if ($shouldOrNot !== "not") {
			foreach (['old', 'new'] as $davVersion) {
				foreach (["{$davVersion}dav-regular", "{$davVersion}dav-{$davVersion}chunking"] as $suffix) {
					if ($suffix !== $exceptChunkingSuffix) {
						$this->checkFileOrFolderExistsForUser(
							$user,
							'file',
							"$destination-$suffix"
						);
					}
				}
			}
		} else {
			foreach (['old', 'new'] as $davVersion) {
				foreach (["{$davVersion}dav-regular", "{$davVersion}dav-{$davVersion}chunking"] as $suffix) {
					if ($suffix !== $exceptChunkingSuffix) {
						$this->checkFileOrFolderDoesNotExistsForUser(
							$user,
							'file',
							"$destination-$suffix"
						);
					}
				}
			}
		}
	}

	/**
	 * @Given user :user has uploaded file :destination of size :bytes bytes
	 *
	 * @param string $user
	 * @param string $destination
	 * @param string $bytes
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userHasUploadedFileToOfSizeBytes(string $user, string $destination, string $bytes): void {
		$user = $this->getActualUsername($user);
		$filename = "filespecificSize.txt";
		$this->createLocalFileOfSpecificSize($filename, $bytes, 'a');
		Assert::assertFileExists($this->workStorageDirLocation() . $filename);
		$response = $this->uploadFile($user, $this->temporaryStorageSubfolderName() . "/$filename", $destination);
		$expectedElements = new TableNode([["$destination"]]);
		$this->checkElementList($user, $expectedElements);
		$this->theHTTPStatusCodeShouldBe([201, 204], '', $response);
	}

	/**
	 * @Given user :user has uploaded file :destination ending with :text of size :bytes bytes
	 *
	 * @param string $user
	 * @param string $destination
	 * @param string $text
	 * @param string $bytes
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userHasUploadedFileToEndingWithOfSizeBytes(
		string $user,
		string $destination,
		string $text,
		string $bytes
	): void {
		$filename = "filespecificSize.txt";
		$this->createLocalFileOfSpecificSize($filename, $bytes, $text);
		Assert::assertFileExists($this->workStorageDirLocation() . $filename);
		$response = $this->uploadFile($user, $this->temporaryStorageSubfolderName() . "/$filename", $destination);
		$this->theHTTPStatusCodeShouldBeBetween(200, 299, $response);
		$this->removeFile($this->workStorageDirLocation(), $filename);
		$expectedElements = new TableNode([["$destination"]]);
		$this->checkElementList($user, $expectedElements);
	}

	/**
	 * @param string $user
	 * @param string|null $content
	 * @param string $destination
	 * @param string $spaceId
	 * @param bool|null $isGivenStep
	 *
	 * @return ResponseInterface
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function uploadFileWithContent(
		string $user,
		?string $content,
		string $destination,
		?string $spaceId = null,
		?bool $isGivenStep = false,
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$this->pauseUploadDelete();

		if (\str_starts_with($destination, "Shares/")
			&& $this->getDavPathVersion() === WebDavHelper::DAV_VERSION_SPACES
		) {
			$spaceId = $this->spacesContext->getSpaceIdByName($user, "Shares");
			$destination = \str_replace("Shares/", "", $destination);
		}
		$response = $this->makeDavRequest(
			$user,
			"PUT",
			$destination,
			[],
			$content,
			$spaceId,
			"files",
			null,
			false,
			null,
			[],
			null,
			$isGivenStep,
		);
		$this->lastUploadDeleteTime = \time();
		return $response;
	}

	/**
	 * @When user :user uploads file with content :content to :destination using the WebDAV API
	 *
	 * @param string $user
	 * @param string|null $content
	 * @param string $destination
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userUploadsAFileWithContentTo(
		string $user,
		?string $content,
		string $destination
	): void {
		$response = $this->uploadFileWithContent($user, $content, $destination);
		$this->setResponse($response);
		$this->pushToLastHttpStatusCodesArray();
	}

	/**
	 * @When /^user "([^"]*)" uploads the following files with content "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string|null $content
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userUploadsFollowingFilesWithContentTo(
		string $user,
		?string $content,
		TableNode $table
	): void {
		$this->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		foreach ($paths as $destination) {
			$response = $this->uploadFileWithContent($user, $content, $destination["path"]);
			$this->setResponse($response);
			$this->pushToLastStatusCodesArrays();
		}
	}

	/**
	 * @When user :user uploads file :source to :destination with mtime :mtime using the WebDAV API
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $mtime Time in human-readable format is taken as input which is converted into milliseconds that is used by API
	 * @param bool|null $isGivenStep
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userUploadsFileToWithMtimeUsingTheWebdavApi(
		string $user,
		string $source,
		string $destination,
		string $mtime,
		?bool $isGivenStep = false
	): void {
		$mtime = new DateTime($mtime);
		$mtime = $mtime->format('U');
		$user = $this->getActualUsername($user);
		$this->response = UploadHelper::upload(
			$this->getBaseUrl(),
			$user,
			$this->getPasswordForUser($user),
			$this->acceptanceTestsDirLocation() . $source,
			$destination,
			$this->getStepLineRef(),
			["X-OC-Mtime" => $mtime],
			$this->getDavPathVersion(),
			false,
			1,
			$isGivenStep
		);
	}

	/**
	 * @Given user :user has uploaded file :source to :destination with mtime :mtime
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $mtime Time in human-readable format is taken as input which is converted into milliseconds that is used by API
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userHasUploadedFileToWithMtimeUsingTheWebdavApi(
		string $user,
		string $source,
		string $destination,
		string $mtime
	): void {
		$mtime = new DateTime($mtime);
		$mtime = $mtime->format('U');
		$user = $this->getActualUsername($user);
		$response = UploadHelper::upload(
			$this->getBaseUrl(),
			$user,
			$this->getPasswordForUser($user),
			$this->acceptanceTestsDirLocation() . $source,
			$destination,
			$this->getStepLineRef(),
			["X-OC-Mtime" => $mtime],
			$this->getDavPathVersion(),
			false,
			1,
			true
		);
		$this->theHTTPStatusCodeShouldBe(
			["201", "204"],
			"",
			$response
		);
	}

	/**
	 * @Then as :user the mtime of the file :resource should be :mtime
	 *
	 * @param string $user
	 * @param string $resource
	 * @param string $mtime
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theMtimeOfTheFileShouldBe(
		string $user,
		string $resource,
		string $mtime
	): void {
		$user = $this->getActualUsername($user);
		$password = $this->getPasswordForUser($user);
		$baseUrl = $this->getBaseUrl();
		$mtime = new DateTime($mtime);
		$mtime = $mtime->format('U');
		Assert::assertEquals(
			$mtime,
			WebDavHelper::getMtimeOfResource(
				$user,
				$password,
				$baseUrl,
				$resource,
				$this->getStepLineRef(),
				$this->getDavPathVersion()
			)
		);
	}

	/**
	 * @Given user :user has uploaded file with content :content to :destination
	 *
	 * @param string $user
	 * @param string|null $content
	 * @param string $destination
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userHasUploadedAFileWithContentTo(
		string $user,
		?string $content,
		string $destination
	): array {
		$user = $this->getActualUsername($user);
		$response = $this->uploadFileWithContent($user, $content, $destination, null, true);
		$this->theHTTPStatusCodeShouldBe(
			["201", "204"],
			"HTTP status code was not 201 or 204 while trying to upload file '$destination' for user '$user'",
			$response
		);
		return $response->getHeader('oc-fileid');
	}

	/**
	 * @Given /^user "([^"]*)" has uploaded the following files with content "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string|null $content
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userHasUploadedFollowingFilesWithContent(
		string $user,
		?string $content,
		TableNode $table
	): void {
		$this->verifyTableNodeColumns($table, ["path"]);
		$files = $table->getHash();

		foreach ($files as $destination) {
			$destination = $destination['path'];
			$user = $this->getActualUsername($user);
			$response = $this->uploadFileWithContent($user, $content, $destination, null, true);
			$this->theHTTPStatusCodeShouldBe(
				["201", "204"],
				"HTTP status code was not 201 or 204 while trying to upload file '$destination' for user '$user'",
				$response
			);
		}
	}

	/**
	 * @When /^user "([^"]*)" downloads the following files using the WebDAV API$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userDownloadsFollowingFiles(
		string $user,
		TableNode $table
	): void {
		$this->verifyTableNodeColumns($table, ["path"]);
		$files = $table->getHash();
		$this->emptyLastHTTPStatusCodesArray();
		foreach ($files as $fileName) {
			$response = $this->downloadFileAsUserUsingPassword($user, $fileName["path"]);
			$this->setResponse($response);
			$this->pushToLastStatusCodesArrays();
		}
	}

	/**
	 * @When user :user uploads a file with content :content and mtime :mtime to :destination using the WebDAV API
	 *
	 * @param string $user
	 * @param string|null $content
	 * @param string $mtime
	 * @param string $destination
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userUploadsAFileWithContentAndMtimeTo(
		string $user,
		?string $content,
		string $mtime,
		string $destination
	): void {
		$user = $this->getActualUsername($user);
		$mtime = new DateTime($mtime);
		$mtime = $mtime->format('U');
		$response = $this->makeDavRequest(
			$user,
			"PUT",
			$destination,
			["X-OC-Mtime" => $mtime],
			$content
		);
		$this->setResponse($response);
	}

	/**
	 * @param string $user
	 * @param string $checksum
	 * @param string|null $content
	 * @param string $destination
	 * @param bool|null $isGivenStep
	 * @param string|null $spaceId
	 *
	 * @return ResponseInterface
	 */
	public function uploadFileWithChecksumAndContent(
		string $user,
		string $checksum,
		?string $content,
		string $destination,
		?bool $isGivenStep = false,
		?string $spaceId = null
	): ResponseInterface {
		$this->pauseUploadDelete();
		$response = $this->makeDavRequest(
			$user,
			"PUT",
			$destination,
			['OC-Checksum' => $checksum],
			$content,
			$spaceId,
			"files",
			null,
			false,
			null,
			[],
			null,
			$isGivenStep
		);
		$this->lastUploadDeleteTime = \time();
		return $response;
	}

	/**
	 * @When user :user uploads file with checksum :checksum and content :content to :destination using the WebDAV API
	 *
	 * @param string $user
	 * @param string $checksum
	 * @param string $content
	 * @param string $destination
	 *
	 * @return void
	 */
	public function userUploadsAFileWithChecksumAndContentTo(
		string $user,
		string $checksum,
		string $content,
		string $destination
	): void {
		$response = $this->uploadFileWithChecksumAndContent($user, $checksum, $content, $destination);
		$this->setResponse($response);
	}

	/**
	 * @Given user :user has uploaded file with checksum :checksum and content :content to :destination
	 *
	 * @param string $user
	 * @param string $checksum
	 * @param string|null $content
	 * @param string $destination
	 *
	 * @return void
	 */
	public function userHasUploadedAFileWithChecksumAndContentTo(
		string $user,
		string $checksum,
		?string $content,
		string $destination
	): void {
		$response = $this->uploadFileWithChecksumAndContent(
			$user,
			$checksum,
			$content,
			$destination,
			true
		);
		$this->theHTTPStatusCodeShouldBe(
			["201", "204"],
			"HTTP status code was not 201 or 204 while trying to upload file with checksum "
			. "'$checksum' to '$destination' for user '$user'",
			$response
		);
	}

	/**
	 * @Then /^user "([^"]*)" should be able to delete (file|folder|entry) "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $entry
	 * @param string $source
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldBeAbleToDeleteEntry(string $user, string $entry, string $source): void {
		$user = $this->getActualUsername($user);
		$this->checkFileOrFolderExistsForUser($user, $entry, $source);
		$this->deleteFile($user, $source);
		$this->checkFileOrFolderDoesNotExistsForUser($user, $entry, $source);
	}

	/**
	 * @Then /^user "([^"]*)" should not be able to delete (file|folder|entry) "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $entry
	 * @param string $source
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theUserShouldNotBeAbleToDeleteEntry(string $user, string $entry, string $source): void {
		$this->checkFileOrFolderExistsForUser($user, $entry, $source);
		$this->deleteFile($user, $source);
		$this->checkFileOrFolderExistsForUser($user, $entry, $source);
	}

	/**
	 * @param string $user
	 * @param string $resource
	 *
	 * @return void
	 */
	public function deleteFile(string $user, string $resource): ResponseInterface {
		$user = $this->getActualUsername($user);
		$this->pauseUploadDelete();
		$response = $this->makeDavRequest($user, 'DELETE', $resource, []);
		$this->lastUploadDeleteTime = \time();
		return $response;
	}

	/**
	 * @When user :user deletes file/folder :resource using the WebDAV API
	 *
	 * @param string $user
	 * @param string $resource
	 *
	 * @return void
	 */
	public function userDeletesFile(string $user, string $resource): void {
		$response = $this->deleteFile($user, $resource);
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @When user :user deletes file :filename from space :space using file-id :fileId
	 *
	 * @param string $user
	 * @param string $filename
	 * @param string $space
	 * @param string $fileId
	 *
	 * @return void
	 */
	public function userDeletesFileFromSpaceUsingFileIdPath(
		string $user,
		string $filename,
		string $space,
		string $fileId
	): void {
		$baseUrl = $this->getBaseUrl();
		$davPath = WebDavHelper::getDavPath($this->getDavPathVersion());
		$user = $this->getActualUsername($user);
		$password =  $this->getPasswordForUser($user);
		$fullUrl = "$baseUrl/$davPath/$fileId";
		$response = HttpRequestHelper::sendRequest(
			$fullUrl,
			null,
			'DELETE',
			$user,
			$password
		);
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @Given user :user has updated a file with content :content using file-id :fileId
	 *
	 * @param string $user
	 * @param string $content
	 * @param string $fileId
	 *
	 * @return void
	 */
	public function userHasUpdatedAFileWithContentUsingFileId(string $user, string $content, string $fileId): void {
		$baseUrl = $this->getBaseUrl();
		$sourceDavPath = WebdavHelper::getDavPath($this->getDavPathVersion());
		$fullUrl = "$baseUrl/$sourceDavPath/$fileId";
		$response = HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->getStepLineRef(),
			'PUT',
			$user,
			$this->getPasswordForUser($user),
			null,
			$content
		);
		$this->theHTTPStatusCodeShouldBe('204', '', $response);
	}

	/**
	 * @Given /^user "([^"]*)" has deleted (?:file|folder|entity) "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $resource
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userHasDeletedResource(string $user, string $resource): void {
		$user = $this->getActualUsername($user);
		$response = $this->deleteFile($user, $resource);
		// If the file or folder was there and got deleted then we get a 204
		// That is good and the expected status
		// If the file or folder was already not there then we get a 404
		// That is not expected. Scenarios that use "Given user has deleted..."
		// should only be using such steps when it is a file that exists and needs
		// to be deleted.

		$this->theHTTPStatusCodeShouldBe(
			["204"],
			"HTTP status code was not 204 while trying to delete resource '$resource' for user '$user'",
			$response
		);
	}

	/**
	 * @Given /^user "([^"]*)" has deleted the following (?:files|folders|resources)$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userHasDeletedFollowingFiles(string $user, TableNode $table): void {
		$this->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		foreach ($paths as $file) {
			$file = $file["path"];
			$user = $this->getActualUsername($user);
			$response = $this->deleteFile($user, $file);
			$this->theHTTPStatusCodeShouldBe(
				["204"],
				"HTTP status code was not 204 while trying to delete resource '$file' for user '$user'",
				$response
			);
		}
	}

	/**
	 * @When /^user "([^"]*)" deletes the following (?:files|folders)$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userDeletesFollowingFiles(string $user, TableNode $table): void {
		$user = $this->getActualUsername($user);
		$this->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		foreach ($paths as $file) {
			$response = $this->deleteFile($user, $file["path"]);
			$this->setResponse($response);
			$this->pushToLastStatusCodesArrays();
			$this->pushToLastStatusCodesArrays();
		}
	}

	/**
	 * @When /^user "([^"]*)" deletes these (?:files|folders|entries) without delays using the WebDAV API$/
	 *
	 * @param string $user
	 * @param TableNode $table of files or folders to delete
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userDeletesFilesFoldersWithoutDelays(string $user, TableNode $table): void {
		$user = $this->getActualUsername($user);
		$this->verifyTableNodeColumnsCount($table, 1);
		foreach ($table->getTable() as $entry) {
			$entryName = $entry[0];
			$this->response = $this->makeDavRequest($user, 'DELETE', $entryName, []);
			$this->pushToLastStatusCodesArrays();
		}
		$this->lastUploadDeleteTime = \time();
	}

	/**
	 * @When user :user creates folder :destination using the WebDAV API
	 *
	 * @param string $user
	 * @param string $destination
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userCreatesFolder(string $user, string $destination): void {
		$response = $this->createFolder($user, $destination);
		$this->setResponse($response);
	}

	/**
	 * @Given user :user has created folder :destination
	 *
	 * @param string $user
	 * @param string $destination
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function userHasCreatedFolder(string $user, string $destination): void {
		$user = $this->getActualUsername($user);
		$response = $this->createFolder($user, $destination, true);
		$this->theHTTPStatusCodeShouldBe(
			["201", "204"],
			"HTTP status code was not 201 or 204 while trying to create folder '$destination' for user '$user'",
			$response
		);
	}

	/**
	 * @Given admin has created folder :destination
	 *
	 * @param string $destination
	 *
	 * @return void
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function adminHasCreatedFolder(string $destination): void {
		$admin = $this->getAdminUsername();
		Assert::assertEquals(
			"admin",
			$admin,
			__METHOD__ . "The provided user is not admin but '" . $admin . "'"
		);
		$response = $this->createFolder($admin, $destination, true);
		$this->theHTTPStatusCodeShouldBe(
			["201", "204"],
			"HTTP status code was not 201 or 204 while trying to create folder '$destination' for admin '$admin'",
			$response
		);
		$this->adminResources[] = $destination;
	}

	/**
	 * @Given /^user "([^"]*)" has created the following folders$/
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userHasCreatedFollowingFolders(string $user, TableNode $table): void {
		$this->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		foreach ($paths as $path) {
			$destination = $path["path"];
			$user = $this->getActualUsername($user);
			$response = $this->createFolder($user, $destination, true);
			$this->theHTTPStatusCodeShouldBe(
				["201", "204"],
				"HTTP status code was not 201 or 204 while trying to create folder '$destination' for user '$user'",
				$response
			);
		}
	}

	/**
	 * @Then user :user should be able to create folder :destination
	 *
	 * @param string $user
	 * @param string $destination
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldBeAbleToCreateFolder(string $user, string $destination): void {
		$user = $this->getActualUsername($user);
		$response = $this->createFolder($user, $destination, true);
		$this->theHTTPStatusCodeShouldBe(
			["201", "204"],
			"HTTP status code was not 201 or 204 while trying to create folder '$destination' for user '$user'",
			$response
		);
		$this->checkFileOrFolderExistsForUser(
			$user,
			"folder",
			$destination
		);
	}

	/**
	 * @Then user :user should be able to create folder :destination using password :password
	 *
	 * @param string $user
	 * @param string $destination
	 * @param string $password
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldBeAbleToCreateFolderUsingPassword(
		string $user,
		string $destination,
		string $password
	): void {
		$user = $this->getActualUsername($user);
		$response = $this->createFolder($user, $destination, true, $password);
		$this->theHTTPStatusCodeShouldBe(
			["201", "204"],
			"HTTP status code was not 201 or 204 while trying to create folder '$destination' for user '$user'",
			$response
		);
		$this->checkFileOrFolderExistsForUser(
			$user,
			"folder",
			$destination
		);
	}

	/**
	 * @Then user :user should not be able to create folder :destination
	 *
	 * @param string $user
	 * @param string $destination
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldNotBeAbleToCreateFolder(string $user, string $destination): void {
		$user = $this->getActualUsername($user);
		$response = $this->createFolder($user, $destination);
		$this->theHTTPStatusCodeShouldBeBetween(400, 499, $response);
		$this->checkFileOrFolderDoesNotExistsForUser(
			$user,
			"folder",
			$destination
		);
	}

	/**
	 * @Then user :user should not be able to create folder :destination using password :password
	 *
	 * @param string $user
	 * @param string $destination
	 * @param string $password
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldNotBeAbleToCreateFolderUsingPassword(
		string $user,
		string $destination,
		string $password
	): void {
		$user = $this->getActualUsername($user);
		$response = $this->createFolder($user, $destination, false, $password);
		$this->theHTTPStatusCodeShouldBeBetween(400, 499, $response);
		$this->checkFileOrFolderDoesNotExistsForUser(
			$user,
			"folder",
			$destination
		);
	}
	/**
	 * Old style chunking upload
	 *
	 * @When user :user uploads the following :total chunks to :file with old chunking and using the WebDAV API
	 *
	 * @param string $user
	 * @param string $total
	 * @param string $file
	 * @param TableNode $chunkDetails table of 2 columns, chunk number and chunk
	 *                                content with column headings, e.g.
	 *                                | number | content                 |
	 *                                | 1      | first data              |
	 *                                | 2      | followed by second data |
	 *                                Chunks may be numbered out-of-order if desired.
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userUploadsTheFollowingTotalChunksUsingOldChunking(
		string $user,
		string $total,
		string $file,
		TableNode $chunkDetails
	): void {
		$this->verifyTableNodeColumns($chunkDetails, ['number', 'content']);
		foreach ($chunkDetails->getHash() as $chunkDetail) {
			$chunkNumber = (int)$chunkDetail['number'];
			$chunkContent = $chunkDetail['content'];
			$this->setResponse($this->userUploadChunkedFile($user, $chunkNumber, (int)$total, $chunkContent, $file));
		}
	}

	/**
	 * Old style chunking upload
	 *
	 * @When user :user uploads the following chunks to :file with old chunking and using the WebDAV API
	 *
	 * @param string $user
	 * @param string $file
	 * @param TableNode $chunkDetails table of 2 columns, chunk number and chunk
	 *                                content with column headings, e.g.
	 *                                | number | content                 |
	 *                                | 1      | first data              |
	 *                                | 2      | followed by second data |
	 *                                Chunks may be numbered out-of-order if desired.
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userUploadsTheFollowingChunksUsingOldChunking(
		string $user,
		string $file,
		TableNode $chunkDetails
	): void {
		$total = (string) \count($chunkDetails->getHash());
		$this->verifyTableNodeColumns($chunkDetails, ['number', 'content']);
		foreach ($chunkDetails->getHash() as $chunkDetail) {
			$chunkNumber = (int)$chunkDetail['number'];
			$chunkContent = $chunkDetail['content'];
			$this->setResponse($this->userUploadChunkedFile($user, $chunkNumber, (int)$total, $chunkContent, $file));
		}
	}

	/**
	 * Old style chunking upload
	 *
	 * @param string $user
	 * @param int $num
	 * @param int $total
	 * @param string|null $data
	 * @param string $destination
	 * @param bool|null $isGivenStep
	 *
	 * @return ResponseInterface
	 */
	public function userUploadChunkedFile(
		string $user,
		int $num,
		int $total,
		?string $data,
		string $destination,
		?bool $isGivenStep = false
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$num -= 1;
		$file = "$destination-chunking-42-$total-$num";
		$this->pauseUploadDelete();
		$response = $this->makeDavRequest(
			$user,
			'PUT',
			$file,
			['OC-Chunked' => '1'],
			$data,
			null,
			"files",
			null,
			false,
			null,
			[],
			null,
			$isGivenStep
		);
		$this->lastUploadDeleteTime = \time();
		return $response;
	}

	/**
	 * New style chunking upload
	 *
	 * @param string $user
	 * @param string $type "asynchronously" or empty
	 * @param string $file
	 * @param TableNode $chunkDetails table of 2 columns, chunk number and chunk
	 *                                content with column headings, e.g.
	 *                                | number | content            |
	 *                                | 1      | first data         |
	 *                                | 2      | second data        |
	 *                                Chunks may be numbered out-of-order if desired.
	 * @param bool|null $isGivenStep
	 *
	 * @return void
	 * @throws Exception
	 */
	public function uploadTheFollowingChunksUsingNewChunking(
		string $user,
		string $type,
		string $file,
		TableNode $chunkDetails,
		?bool $isGivenStep = false
	): void {
		$user = $this->getActualUsername($user);
		$async = false;
		if ($type === "asynchronously") {
			$async = true;
		}
		$this->verifyTableNodeColumns($chunkDetails, ["number", "content"]);
		$this->userUploadsChunksUsingNewChunking(
			$user,
			$file,
			'chunking-42',
			$chunkDetails->getHash(),
			$async,
			$isGivenStep
		);
	}

	/**
	 * New style chunking upload
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $chunkingId
	 * @param array $chunkDetails of chunks of the file. Each array entry is
	 *                            itself an array of 2 items:
	 *                            [number] the chunk number
	 *                            [content] data content of the chunk
	 *                            Chunks may be numbered out-of-order if desired.
	 * @param bool $async use asynchronous MOVE at the end or not
	 * @param bool $isGivenStep
	 *
	 * @return void
	 */
	public function userUploadsChunksUsingNewChunking(
		string $user,
		string $file,
		string $chunkingId,
		array $chunkDetails,
		bool $async = false,
		bool $isGivenStep = false
	): void {
		$this->pauseUploadDelete();
		if ($isGivenStep) {
			$response = $this->userCreateANewChunkingUploadWithId($user, $chunkingId, true);
			$this->theHTTPStatusCodeShouldBeBetween(200, 299, $response);
		} else {
			$this->setResponse($this->userCreateANewChunkingUploadWithId($user, $chunkingId));
		}
		foreach ($chunkDetails as $chunkDetail) {
			$chunkNumber = (int)$chunkDetail['number'];
			$chunkContent = $chunkDetail['content'];
			if ($isGivenStep) {
				$response = $this->userUploadNewChunkFileOfWithToId(
					$user,
					$chunkNumber,
					$chunkContent,
					$chunkingId,
					true
				);
				$this->theHTTPStatusCodeShouldBeBetween(200, 299, $response);
			} else {
				$response = $this->userUploadNewChunkFileOfWithToId($user, $chunkNumber, $chunkContent, $chunkingId);
				$this->setResponse($response);
				$this->pushToLastStatusCodesArrays();
			}
		}
		$headers = [];
		if ($async === true) {
			$headers = ['OC-LazyOps' => 'true'];
		}
		$response = $this->moveNewDavChunkToFinalFile($user, $chunkingId, $file, $headers, $isGivenStep);
		if ($isGivenStep) {
			$this->theHTTPStatusCodeShouldBeBetween(200, 299, $response);
		} else {
			$this->setResponse($response);
		}
		$this->lastUploadDeleteTime = \time();
	}

	/**
	 *
	 * @param string $user
	 * @param string $id
	 * @param bool|null $isGivenStep
	 *
	 * @return ResponseInterface
	 */
	public function userCreateANewChunkingUploadWithId(
		string $user,
		string $id,
		?bool $isGivenStep = false
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$destination = "/uploads/$user/$id";
		return $this->makeDavRequest(
			$user,
			'MKCOL',
			$destination,
			[],
			null,
			null,
			"uploads",
			null,
			false,
			null,
			[],
			null,
			$isGivenStep
		);
	}

	/**
	 * @param string $user
	 * @param int $num
	 * @param string|null $data
	 * @param string $id
	 * @param bool|null $isGivenStep
	 *
	 * @return ResponseInterface
	 */
	public function userUploadNewChunkFileOfWithToId(
		string $user,
		int $num,
		?string $data,
		string $id,
		?bool $isGivenStep = false
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$destination = "/uploads/$user/$id/$num";
		return $this->makeDavRequest(
			$user,
			'PUT',
			$destination,
			[],
			$data,
			null,
			"uploads",
			null,
			false,
			null,
			[],
			null,
			$isGivenStep
		);
	}

	/**
	 * @param string $user
	 * @param string $id
	 * @param string $type "asynchronously" or empty
	 * @param string $dest
	 *
	 * @return ResponseInterface
	 */
	public function userMoveNewChunkFileWithIdToMychunkedfile(
		string $user,
		string $id,
		string $type,
		string $dest
	): ResponseInterface {
		$headers = [];
		if ($type === "asynchronously") {
			$headers = ['OC-LazyOps' => 'true'];
		}
		return $this->moveNewDavChunkToFinalFile($user, $id, $dest, $headers);
	}

	/**
	 * @When /^user "([^"]*)" moves new chunk file with id "([^"]*)"\s?(asynchronously|) to "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $id
	 * @param string $type "asynchronously" or empty
	 * @param string $dest
	 *
	 * @return void
	 */
	public function userMovesNewChunkFileWithIdToMychunkedfile(
		string $user,
		string $id,
		string $type,
		string $dest
	): void {
		$this->setResponse($this->userMoveNewChunkFileWithIdToMychunkedfile($user, $id, $type, $dest));
	}

	/**
	 * @param string $user
	 * @param string $id
	 * @param string $type "asynchronously" or empty
	 * @param string $dest
	 * @param int $size
	 *
	 * @return ResponseInterface
	 */
	public function userMoveNewChunkFileWithIdToMychunkedfileWithSize(
		string $user,
		string $id,
		string $type,
		string $dest,
		int $size
	): ResponseInterface {
		$headers = ['OC-Total-Length' => $size];
		if ($type === "asynchronously") {
			$headers['OC-LazyOps'] = 'true';
		}
		return $this->moveNewDavChunkToFinalFile(
			$user,
			$id,
			$dest,
			$headers
		);
	}

	/**
	 * @param string $user
	 * @param string $id
	 * @param string $type "asynchronously" or empty
	 * @param string $dest
	 * @param string $checksum
	 *
	 * @return ResponseInterface
	 */
	public function userMoveNewChunkFileWithIdToMychunkedfileWithChecksum(
		string $user,
		string $id,
		string $type,
		string $dest,
		string $checksum
	): ResponseInterface {
		$headers = ['OC-Checksum' => $checksum];
		if ($type === "asynchronously") {
			$headers['OC-LazyOps'] = 'true';
		}
		return $this->moveNewDavChunkToFinalFile(
			$user,
			$id,
			$dest,
			$headers
		);
	}

	/**
	 * @Given /^user "([^"]*)" has moved new chunk file with id "([^"]*)"\s?(asynchronously|) to "([^"]*)" with checksum "([^"]*)"
	 *
	 * @param string $user
	 * @param string $id
	 * @param string $type "asynchronously" or empty
	 * @param string $dest
	 * @param string $checksum
	 *
	 * @return void
	 */
	public function userHasMovedNewChunkFileWithIdToMychunkedfileWithChecksum(
		string $user,
		string $id,
		string $type,
		string $dest,
		string $checksum
	): void {
		$response = $this->userMoveNewChunkFileWithIdToMychunkedfileWithChecksum(
			$user,
			$id,
			$type,
			$dest,
			$checksum
		);
		$this->theHTTPStatusCodeShouldBe("201", "", $response);
	}

	/**
	 * Move chunked new DAV file to final file
	 *
	 * @param string $user user
	 * @param string $id upload id
	 * @param string $destination destination path
	 * @param array $headers extra headers
	 * @param bool|null $isGivenStep
	 *
	 * @return ResponseInterface
	 */
	private function moveNewDavChunkToFinalFile(
		string $user,
		string $id,
		string $destination,
		array $headers,
		?bool $isGivenStep = false
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$source = "/uploads/$user/$id/.file";
		$headers['Destination'] = $this->destinationHeaderValue(
			$user,
			$destination
		);

		return $this->makeDavRequest(
			$user,
			'MOVE',
			$source,
			$headers,
			null,
			null,
			"uploads",
			null,
			false,
			null,
			[],
			null,
			$isGivenStep
		);
	}

	/**
	 * Delete chunked-upload directory
	 *
	 * @param string $user user
	 * @param string $id upload id
	 * @param array $headers extra headers
	 *
	 * @return ResponseInterface
	 */
	private function deleteUpload(string $user, string $id, array $headers): ResponseInterface {
		$source = "/uploads/$user/$id";
		return $this->makeDavRequest(
			$user,
			'DELETE',
			$source,
			$headers,
			null,
			null,
			"uploads"
		);
	}

	/**
	 * URL encodes the given path but keeps the slashes
	 *
	 * @param string $path to encode
	 *
	 * @return string encoded path
	 */
	public function encodePath(string $path): string {
		// slashes need to stay
		// in OpenCloud even brackets are encoded
		return \str_replace('%2F', '/', \rawurlencode($path));
	}

	/**
	 * an unauthenticated client connects to the DAV endpoint using the WebDAV API
	 *
	 * @return ResponseInterface
	 */
	public function connectToDavEndpoint(): ResponseInterface {
		return $this->makeDavRequest(
			null,
			'PROPFIND',
			'',
			[]
		);
	}

	/**
	 * @When an unauthenticated client connects to the DAV endpoint using the WebDAV API
	 *
	 * @return void
	 */
	public function connectingToDavEndpoint(): void {
		$this->setResponse($this->connectToDavEndpoint());
	}

	/**
	 * @Then there should be no duplicate headers
	 *
	 * @return void
	 * @throws Exception
	 */
	public function thereAreNoDuplicateHeaders(): void {
		$headers = $this->response->getHeaders();
		foreach ($headers as $headerName => $headerValues) {
			// if a header has multiple values, they must be different
			if (\count($headerValues) > 1
				&& \count(\array_unique($headerValues)) < \count($headerValues)
			) {
				throw new Exception("Duplicate header found: $headerName");
			}
		}
	}

	/**
	 * @Then the following headers should not be set
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theFollowingHeadersShouldNotBeSet(TableNode $table): void {
		$this->verifyTableNodeColumns(
			$table,
			['header']
		);
		foreach ($table->getColumnsHash() as $header) {
			$headerName = $header['header'];
			$headerValue = $this->response->getHeader($headerName);
			//Note: getHeader returns an empty array if the named header does not exist
			$headerValue0 = $headerValue[0] ?? '';
			Assert::assertEmpty(
				$headerValue,
				"header $headerName should not exist " .
				"but does and is set to $headerValue0"
			);
		}
	}

	/**
	 * @Then the following headers should match these regular expressions
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theFollowingHeadersShouldMatchTheseRegularExpressions(TableNode $table): void {
		$this->headersShouldMatchRegularExpressions($table);
	}

	/**
	 * @param TableNode $table
	 *
	 * @return void
	 */
	public function headersShouldMatchRegularExpressions(TableNode $table): void {
		$this->verifyTableNodeColumnsCount($table, 2);
		foreach ($table->getTable() as $header) {
			$headerName = $header[0];
			$expectedHeaderValue = $header[1];
			$expectedHeaderValue = $this->substituteInLineCodes(
				$expectedHeaderValue,
				null,
				['preg_quote' => ['/']]
			);

			$returnedHeaders = $this->response->getHeader($headerName);
			$returnedHeader = $returnedHeaders[0];
			Assert::assertNotFalse(
				(bool) \preg_match($expectedHeaderValue, $returnedHeader),
				"'$expectedHeaderValue' does not match '$returnedHeader'"
			);
		}
	}

	/**
	 * @Then /^if the HTTP status code was "([^"]*)" then the following headers should match these regular expressions$/
	 *
	 * @param int $statusCode
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function statusCodeShouldMatchTheseRegularExpressions(int $statusCode, TableNode $table): void {
		$actualStatusCode = $this->response->getStatusCode();
		if ($actualStatusCode === $statusCode) {
			$this->headersShouldMatchRegularExpressions($table);
		}
	}

	/**
	 * @Then the following headers should match these regular expressions for user :user
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function headersShouldMatchRegularExpressionsForUser(string $user, TableNode $table): void {
		$this->verifyTableNodeColumnsCount($table, 2);
		$user = $this->getActualUsername($user);
		foreach ($table->getTable() as $header) {
			$headerName = $header[0];
			$expectedHeaderValue = $header[1];
			$expectedHeaderValue = $this->substituteInLineCodes(
				$expectedHeaderValue,
				$user,
				['preg_quote' => ['/']]
			);

			$returnedHeaders = $this->response->getHeader($headerName);
			$returnedHeader = $returnedHeaders[0];
			Assert::assertNotFalse(
				(bool) \preg_match($expectedHeaderValue, $returnedHeader),
				"'$expectedHeaderValue' does not match '$returnedHeader'"
			);
		}
	}

	/**
	 * @When /^user "([^"]*)" deletes everything from folder "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $folder
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userDeletesEverythingInFolder(
		string $user,
		string $folder
	): void {
		$this->deleteEverythingInFolder($user, $folder, false);
	}

	/**
	 * @param string $user
	 * @param string $folder
	 * @param boolean $checkEachDelete
	 *
	 * @return void
	 * @throws Exception
	 */
	public function deleteEverythingInFolder(
		string $user,
		string $folder,
		bool $checkEachDelete = false
	): void {
		$user = $this->getActualUsername($user);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$this->listFolder(
				$user,
				$folder,
				'1'
			)
		);
		$elementList = $responseXmlObject->xpath("//d:response/d:href");
		if (\is_array($elementList) && \count($elementList)) {
			\array_shift($elementList); //don't delete the folder itself
			$davPrefix = "/" . $this->getFullDavFilesPath($user);
			foreach ($elementList as $element) {
				$element = \substr((string)$element, \strlen($davPrefix));
				if ($checkEachDelete) {
					$user = $this->getActualUsername($user);
					$response = $this->deleteFile($user, $element);
					$this->theHTTPStatusCodeShouldBe(
						["204"],
						"HTTP status code was not 204 while trying to delete resource '$element' for user '$user'",
						$response
					);
				} else {
					$this->setResponse($this->deleteFile($user, $element));
					$this->pushToLastStatusCodesArrays();
				}
			}
		}
	}

	/**
	 * @When user :user downloads the preview of :path with width :width and height :height using the WebDAV API
	 * @When user :user tries to download the preview of nonexistent file :path with width :width and height :height using the WebDAV API
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $width
	 * @param string $height
	 *
	 * @return void
	 */
	public function downloadPreviewOfFiles(string $user, string $path, string $width, string $height): void {
		$response = $this->downloadPreviews(
			$user,
			$path,
			null,
			$width,
			$height
		);
		$this->setResponse($response);
	}

	/**
	 * @When user :user tries to download the preview of :path with width :width and height :height and preview set to 0 using the WebDAV API
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $width
	 * @param string $height
	 *
	 * @return void
	 */
	public function userDownloadsThePreviewOfWithPreviewZero(
		string $user,
		string $path,
		string $width,
		string $height
	) {
		$user = $this->getActualUsername($user);
		$urlParameter = [
			'x' => $width,
			'y' => $height,
			'preview' => '0'
		];
		$response = $this->makeDavRequest(
			$user,
			"GET",
			$path,
			[],
			null,
			null,
			"files",
			null,
			false,
			null,
			$urlParameter,
		);
		$this->setResponse($response);
	}

	/**
	 * @When user :user downloads the preview of shared resource :path with width :width and height :height using the WebDAV API
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $width
	 * @param string $height
	 *
	 * @return void
	 */
	public function userDownloadsThePreviewOfSharedResourceWithWidthAndHeightUsingTheWebdavApi(
		string $user,
		string $path,
		string $width,
		string $height
	): void {
		if ($this->getDavPathVersion() === WebDavHelper::DAV_VERSION_SPACES) {
			$this->setResponse($this->downloadSharedFilePreview($user, $path, $width, $height));
		} else {
			$this->setResponse($this->downloadPreviews($user, $path, null, $width, $height));
		}
	}

	/**
	 * @When user :user downloads the preview of :path with width :width and height :height and processor :processor using the WebDAV API
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $width
	 * @param string $height
	 * @param string $processor
	 *
	 * @return void
	 */
	public function userDownloadsThePreviewOfWithWidthHeightProcessorUsingWebDAVAPI(
		string $user,
		string $path,
		string $width,
		string $height,
		string $processor
	): void {
		$user = $this->getActualUsername($user);
		$urlParameter = [
			'x' => $width,
			'y' => $height,
			'preview' => '1',
			'processor' => $processor
		];
		$response = $this->makeDavRequest(
			$user,
			"GET",
			$path,
			[],
			null,
			null,
			"files",
			null,
			false,
			null,
			$urlParameter,
		);
		$this->setResponse($response);
	}

	/**
	 * @When user :user downloads the preview of federated share image :path with width :width and height :height using the WebDAV API
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $width
	 * @param string $height
	 *
	 * @return void
	 */
	public function userDownloadsThePreviewOfFederatedShareImageWithWidthHeightUsingWebDAVAPI(
		string $user,
		string $path,
		string $width,
		string $height
	): void {
		$user = $this->getActualUsername($user);
		$urlParameter = [
			'x' => $width,
			'y' => $height,
			'preview' => '1'
		];
		$spaceId = $this->spacesContext->getSharesRemoteItemId($user, $path);
		$this->setResponse(
			$this->makeDavRequest(
				$user,
				"GET",
				$path,
				[],
				null,
				$spaceId,
				"files",
				null,
				false,
				null,
				$urlParameter,
			)
		);
	}

	/**
	 * @Given user :user has downloaded the preview of shared resource :path with width :width and height :height
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $width
	 * @param string $height
	 *
	 * @return void
	 */
	public function userHasDownloadedThePreviewOfSharedResourceWithWidthAndHeight(
		string $user,
		string $path,
		string $width,
		string $height
	): void {
		if ($this->getDavPathVersion() === WebDavHelper::DAV_VERSION_SPACES) {
			$response = $this->downloadSharedFilePreview($user, $path, $width, $height);
		} else {
			$response = $this->downloadPreviews($user, $path, null, $width, $height);
		}
		$this->setResponse($response);
		$this->theHTTPStatusCodeShouldBe(200, '', $response);
		$this->checkImageDimensions($width, $height);
		$response->getBody()->rewind();
		// save response to user response dictionary for further comparisons
		$this->setFilePreviewContent($user, $response->getBody()->getContents());
	}

	/**
	 * @Then as user :user the preview of shared resource :path with width :width and height :height should have been changed
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $width
	 * @param string $height
	 *
	 * @return void
	 */
	public function asUserThePreviewOfSharedResourceWithWidthAndHeightShouldHaveBeenChanged(
		string $user,
		string $path,
		string $width,
		string $height
	): void {
		if ($this->getDavPathVersion() === WebDavHelper::DAV_VERSION_SPACES) {
			$response = $this->downloadSharedFilePreview($user, $path, $width, $height);
		} else {
			$response = $this->downloadPreviews($user, $path, null, $width, $height);
		}
		$this->setResponse($response);
		$this->theHTTPStatusCodeShouldBe(200, '', $response);
		$newResponseBodyContents = $this->response->getBody()->getContents();
		Assert::assertNotEquals(
			$newResponseBodyContents,
			// different users can download files before and after an update is made to a file
			// previous response content is fetched from the user response body content array entry for that user
			$this->getFilePreviewContent($user),
			__METHOD__ . " previous and current previews content is same but expected to be different",
		);
		// update the saved content for the next comparison
		$this->setFilePreviewContent($user, $newResponseBodyContents);
	}

	/**
	 * @When user :user uploads file with content :content to shared resource :destination using the WebDAV API
	 *
	 * @param string $user
	 * @param string $content
	 * @param string $destination
	 *
	 * @return void
	 */
	public function userUploadsFileWithContentSharedResourceToUsingTheWebdavApi(
		string $user,
		string $content,
		string $destination
	): void {
		$this->setResponse($this->uploadFileWithContent($user, $content, $destination));
	}

	/**
	 * @param string $user
	 * @param string $path
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function getSharesMountPath(
		string $user,
		string $path
	): string {
		$user = $this->getActualUsername($user);
		$path = trim($path, "/");
		$pathArray = explode("/", $path);
		$sharedFolder = $pathArray[0] === "Shares" ? $pathArray[1] : $pathArray[0];

		$shareMountId = GraphHelper::getShareMountId(
			$this->getBaseUrl(),
			$this->getStepLineRef(),
			$user,
			$this->getPasswordForUser($user),
			$sharedFolder
		);

		$path = \array_slice($pathArray, array_search($sharedFolder, $pathArray) + 1);
		$path = \implode("/", $path);

		return "$shareMountId/$path";
	}

	/**
	 * @param string $user
	 * @param string $path
	 * @param string|null $width
	 * @param string|null $height
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function downloadSharedFilePreview(
		string $user,
		string $path,
		?string $width = null,
		?string $height = null
	): ResponseInterface {
		if ($width !== null && $height !== null) {
			$urlParameter = [
				'x' => $width,
				'y' => $height,
				'forceIcon' => '0',
				'preview' => '1'
			];
			$urlParameter = \http_build_query($urlParameter, '', '&');
		} else {
			$urlParameter = null;
		}

		$davPathVersion = $this->getDavPathVersion();
		$suffixPath = $user;
		if ($davPathVersion === WebDavHelper::DAV_VERSION_SPACES) {
			$suffixPath = $this->getSharesMountPath($user, $path);
			$path = "";
		}
		$path = \ltrim($path, "/");
		$davPath = WebDavHelper::getDavPath($this->getDavPathVersion(), $suffixPath);
		$davPath = \rtrim($davPath, "/");
		$fullUrl = $this->getBaseUrl() . "/$davPath/$path?$urlParameter";

		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->getStepLineRef(),
			'GET',
			$user,
			$this->getPasswordForUser($user)
		);
	}

	/**
	 * @When user :user downloads the preview of :path of :ofUser with width :width and height :height using the WebDAV API
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $ofUser
	 * @param string $width
	 * @param string $height
	 *
	 * @return void
	 */
	public function downloadPreviewOfOtherUser(
		string $user,
		string $path,
		string $ofUser,
		string $width,
		string $height
	): void {
		$response = $this->downloadPreviews(
			$ofUser,
			$path,
			$user,
			$width,
			$height
		);
		$this->setResponse($response);
	}

	/**
	 * @Then the downloaded image should be :width pixels wide and :height pixels high
	 *
	 * @param string $width
	 * @param string $height
	 *
	 * @return void
	 */
	public function imageDimensionsShouldBe(string $width, string $height): void {
		$this->checkImageDimensions($width, $height);
	}

	/**
	 * @param string $width
	 * @param string $height
	 * @param ResponseInterface|null $response
	 *
	 * @return void
	 */
	public function checkImageDimensions(string $width, string $height, ?ResponseInterface $response = null): void {
		$response = $response ?? $this->getResponse();
		$response->getBody()->rewind();
		$responseBodyContent = $response->getBody()->getContents();
		$size = \getimagesizefromstring($responseBodyContent);
		Assert::assertNotFalse($size, "could not get size of image");
		Assert::assertEquals($width, $size[0], "width not as expected");
		Assert::assertEquals($height, $size[1], "height not as expected");
	}

	/**
	 * @Then the downloaded preview content should match with :preview fixtures preview content
	 *
	 * @param string $filename relative path from fixtures directory
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theDownloadedPreviewContentShouldMatchWithFixturesPreviewContentFor(string $filename): void {
		$expectedPreview = \file_get_contents(__DIR__ . "/../fixtures/" . $filename);
		$this->getResponse()->getBody()->rewind();
		$responseBodyContent = $this->getResponse()->getBody()->getContents();
		Assert::assertEquals($expectedPreview, $responseBodyContent);
	}

	/**
	 * @Given user :user has downloaded the preview of :path with width :width and height :height
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $width
	 * @param string $height
	 *
	 * @return void
	 */
	public function userDownloadsThePreviewOfWithWidthAndHeight(
		string $user,
		string $path,
		string $width,
		string $height
	): void {
		$response = $this->downloadPreviews(
			$user,
			$path,
			null,
			$width,
			$height
		);
		$this->theHTTPStatusCodeShouldBe(200, "", $response);
		$this->checkImageDimensions($width, $height, $response);
		$response->getBody()->rewind();
		// save response to user response dictionary for further comparisons
		$this->setFilePreviewContent($user, $response->getBody()->getContents());
	}

	/**
	 * @Then as user :user the preview of :path with width :width and height :height should have been changed
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $width
	 * @param string $height
	 *
	 * @return void
	 */
	public function asUserThePreviewOfPathWithHeightAndWidthShouldHaveBeenChanged(
		string $user,
		string $path,
		string $width,
		string $height
	): void {
		$response = $this->downloadPreviews(
			$user,
			$path,
			null,
			$width,
			$height
		);
		$this->theHTTPStatusCodeShouldBe(200, "", $response);
		$newResponseBodyContents = $response->getBody()->getContents();
		Assert::assertNotEquals(
			$newResponseBodyContents,
			// different users can download files before and after an update is made to a file
			// previous response content is fetched from the user response body content array entry for that user
			$this->getFilePreviewContent($user),
			__METHOD__ . " previous and current previews content is same but expected to be different",
		);
		// update the saved content for the next comparison
		$this->setFilePreviewContent($user, $newResponseBodyContents);
	}

	/**
	 * @param string $user
	 * @param string $path
	 * @param string $spaceId
	 *
	 * @return string|null
	 */
	public function getFileIdForPath(string $user, string $path, string $spaceId = null): ?string {
		$user = $this->getActualUsername($user);
		try {
			return WebDavHelper::getFileIdForPath(
				$this->getBaseUrl(),
				$user,
				$this->getPasswordForUser($user),
				$path,
				$spaceId,
				$this->getStepLineRef(),
				$this->getDavPathVersion()
			);
		} catch (Exception $e) {
			return null;
		}
	}

	/**
	 * @Given /^user "([^"]*)" has stored id of (?:file|folder) "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $path
	 *
	 * @return void
	 */
	public function userStoresFileIdForPath(string $user, string $path): void {
		$this->storedFileID = $this->getFileIdForPath($user, $path);
	}

	/**
	 * @Then /^user "([^"]*)" (file|folder) "([^"]*)" should have the previously stored id$/
	 *
	 * @param string370 $user
	 * @param string $fileOrFolder
	 * @param string $path
	 *
	 * @return void
	 */
	public function userFileShouldHaveStoredId(string $user, string $fileOrFolder, string $path): void {
		$user = $this->getActualUsername($user);
		$currentFileID = $this->getFileIdForPath($user, $path);
		Assert::assertEquals(
			$currentFileID,
			$this->storedFileID,
			__METHOD__
			. " User '$user' $fileOrFolder '$path' does not have the previously stored id " .
			"'$this->storedFileID', but has '$currentFileID'."
		);
	}

	/**
	 * @Then /^the (?:Cal|Card)?DAV (exception|message|reason) should be "([^"]*)"$/
	 *
	 * @param string $element exception|message|reason
	 * @param string $message
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theDavElementShouldBe(string $element, string $message): void {
		$responseXmlArray = HttpRequestHelper::parseResponseAsXml($this->getResponse());
		WebDavAssert::assertDavResponseElementIs(
			$element,
			$message,
			$responseXmlArray,
			__METHOD__
		);
	}

	/**
	 * @param string $shouldOrNot (not|)
	 * @param TableNode $expectedFiles
	 * @param string|null $user
	 * @param string|null $method
	 * @param string|null $folderpath
	 * @param string|null $spaceId
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function propfindResultShouldContainEntries(
		string $shouldOrNot,
		TableNode $expectedFiles,
		?string $user = null,
		?string $method = 'REPORT',
		?string $folderpath = '',
		?string $spaceId = null
	): void {
		$folderpath = \trim($folderpath, "/");
		$this->verifyTableNodeColumnsCount($expectedFiles, 1);
		$elementRows = $expectedFiles->getRows();
		$should = ($shouldOrNot !== "not");
		foreach ($elementRows as $expectedFile) {
			$resource = $expectedFile[0];
			$resource = $this->substituteInLineCodes($resource, $user);
			if ($resource === '') {
				continue;
			}
			if ($method === "REPORT") {
				$fileFound = $this->findEntryFromSearchResponse(
					$resource,
					false,
					$spaceId,
				);
				if (\is_object($fileFound)) {
					$fileFound = $fileFound->xpath("d:propstat//oc:name");
				}
			} else {
				$fileFound = $this->findEntryFromPropfindResponse(
					$resource,
					$user,
					"files",
					$folderpath,
					$spaceId
				);
			}
			if ($should) {
				Assert::assertNotEmpty(
					$fileFound,
					"response does not contain the entry '$resource'"
				);
			} else {
				Assert::assertFalse(
					$fileFound,
					"response does contain the entry '$resource' but should not"
				);
			}
		}
	}

	/**
	 * @Then /^the (?:propfind|search) result of user "([^"]*)" should (not|)\s?contain these (?:files|entries):$/
	 *
	 * @param string $user
	 * @param string $shouldOrNot (not|)
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 * @throws Exception
	 */
	public function thePropfindResultShouldContainEntries(
		string $user,
		string $shouldOrNot,
		TableNode $expectedFiles
	): void {
		$user = $this->getActualUsername($user);
		$this->propfindResultShouldContainEntries(
			$shouldOrNot,
			$expectedFiles,
			$user
		);
	}

	/**
	 * @Then /^the (?:propfind|search) result of user "([^"]*)" should contain only these (?:files|entries):$/
	 *
	 * @param string $user
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 * @throws Exception
	 */
	public function thePropfindResultShouldContainOnlyEntries(
		string $user,
		TableNode $expectedFiles
	): void {
		$user = $this->getActualUsername($user);

		Assert::assertEquals(
			\count($expectedFiles->getTable()),
			$this->getNumberOfEntriesInPropfindResponse(
				$user
			),
			"The number of elements in the response doesn't matches with expected number of elements"
		);
		$this->propfindResultShouldContainEntries(
			'',
			$expectedFiles,
			$user
		);
	}

	/**
	 * @Then the propfind/search result should contain :numFiles files/entries
	 *
	 * @param int $numFiles
	 *
	 * @return void
	 */
	public function propfindResultShouldContainNumEntries(int $numFiles): void {
		$this->checkIFResponseContainsNumberEntries($numFiles);
	}

	/**
	 * @param integer $numFiles
	 *
	 * @return void
	 */
	public function checkIFResponseContainsNumberEntries(int $numFiles): void {
		$responseXmlArray = HttpRequestHelper::parseResponseAsXml($this->response);

		Assert::assertIsArray(
			$responseXmlArray,
			__METHOD__ . " is not an array"
		);
		Assert::assertArrayHasKey(
			"value",
			$responseXmlArray,
			__METHOD__ . " does not have key 'value'"
		);
		$multistatusResults = $responseXmlArray["value"];
		if ($multistatusResults === null) {
			$multistatusResults = [];
		}
		Assert::assertEquals(
			$numFiles,
			\count($multistatusResults),
			__METHOD__
			. " Expected result to contain '"
			. $numFiles
			. "' files/entries, but got '"
			. \count($multistatusResults)
			. "' files/entries."
		);
	}

	/**
	 * @Then the propfind/search result should contain any :expectedNumber of these files/entries:
	 *
	 * @param integer $expectedNumber
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theSearchResultShouldContainAnyOfTheseEntries(
		int $expectedNumber,
		TableNode $expectedFiles
	): void {
		$this->checkIfSearchResultContainsFiles(
			$this->getCurrentUser(),
			$expectedNumber,
			$expectedFiles
		);
	}

	/**
	 * @Then the propfind/search result of user :user should contain any :expectedNumber of these files/entries:
	 *
	 * @param string $user
	 * @param integer $expectedNumber
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theSearchResultOfUserShouldContainAnyOfTheseEntries(
		string $user,
		int $expectedNumber,
		TableNode $expectedFiles
	): void {
		$this->checkIfSearchResultContainsFiles(
			$user,
			$expectedNumber,
			$expectedFiles
		);
	}

	/**
	 * @param string $user
	 * @param integer $expectedNumber
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 */
	public function checkIfSearchResultContainsFiles(
		string $user,
		int $expectedNumber,
		TableNode $expectedFiles
	): void {
		$user = $this->getActualUsername($user);
		$this->verifyTableNodeColumnsCount($expectedFiles, 1);
		$this->checkIFResponseContainsNumberEntries($expectedNumber);
		$elementRows = $expectedFiles->getColumn(0);
		// Remove any "/" from the front (or back) of the expected values passed
		// into the step. findEntryFromPropfindResponse returns entries without
		// any leading (or trailing) slash
		$expectedEntries = \array_map(
			function ($value) {
				return \trim($value, "/");
			},
			$elementRows
		);
		$resultEntries = $this->findEntryFromSearchResponse();
		foreach ($resultEntries as $resultEntry) {
			Assert::assertContains($resultEntry, $expectedEntries);
		}
	}

	/**
	 * @When user :arg1 lists the resources in :path with depth :depth using the WebDAV API
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $depth
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userListsTheResourcesInPathWithDepthUsingTheWebdavApi(
		string $user,
		string $path,
		string $depth
	): void {
		$response = $this->listFolder(
			$user,
			$path,
			$depth
		);
		$this->setResponse($response);
	}

	/**
	 * @Then the last DAV response for user :user should contain these nodes/elements
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theLastDavResponseShouldContainTheseNodes(string $user, TableNode $table): void {
		$this->verifyTableNodeColumns($table, ["name"]);
		foreach ($table->getHash() as $row) {
			$path = $this->substituteInLineCodes($row['name']);
			$res = $this->findEntryFromPropfindResponse($path, $user);
			Assert::assertNotFalse($res, "expected $path to be in DAV response but was not found");
		}
	}

	/**
	 * @Then the last DAV response for user :user should not contain these nodes/elements
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theLastDavResponseShouldNotContainTheseNodes(string $user, TableNode $table): void {
		$this->verifyTableNodeColumns($table, ["name"]);
		foreach ($table->getHash() as $row) {
			$path = $this->substituteInLineCodes($row['name']);
			$res = $this->findEntryFromPropfindResponse($path, $user);
			Assert::assertFalse($res, "expected $path to not be in DAV response but was found");
		}
	}

	/**
	 * @Then the last public link DAV response should contain these nodes/elements
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theLastPublicDavResponseShouldContainTheseNodes(TableNode $table): void {
		$token = ($this->isUsingSharingNG())
		? $this->shareNgGetLastCreatedLinkShareToken() : $this->getLastCreatedPublicShareToken();
		$this->verifyTableNodeColumns($table, ["name"]);

		foreach ($table->getHash() as $row) {
			$path = $this->substituteInLineCodes($row['name']);
			$res = $this->findEntryFromPropfindResponse($path, $token, "public-files");
			Assert::assertNotFalse($res, "expected $path to be in DAV response but was not found");
		}
	}

	/**
	 * @Then the last public link DAV response should not contain these nodes/elements
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theLastPublicDavResponseShouldNotContainTheseNodes(TableNode $table): void {
		$token = ($this->isUsingSharingNG())
		? $this->shareNgGetLastCreatedLinkShareToken() : $this->getLastCreatedPublicShareToken();
		$this->verifyTableNodeColumns($table, ["name"]);

		foreach ($table->getHash() as $row) {
			$path = $this->substituteInLineCodes($row['name']);
			$res = $this->findEntryFromPropfindResponse($path, $token, "public-files");
			Assert::assertFalse($res, "expected $path to not be in DAV response but was found");
		}
	}

	/**
	 * @When the public lists the resources in the last created public link with depth :depth using the WebDAV API
	 *
	 * @param string $depth
	 *
	 * @return void
	 * @throws Exception
	 */
	public function thePublicListsTheResourcesInTheLastCreatedPublicLinkWithDepthUsingTheWebdavApi(
		string $depth
	): void {
		$token = ($this->isUsingSharingNG())
		? $this->shareNgGetLastCreatedLinkShareToken() : $this->getLastCreatedPublicShareToken();
		$this->setResponse(
			$this->listFolder(
				$token,
				'/',
				$depth,
				null,
				null,
				"public-files"
			)
		);
	}

	/**
	 * @param string|null $user
	 *
	 * @return array
	 */
	public function findEntryFromReportResponse(?string $user): array {
		$responseXmlObj = HttpRequestHelper::getResponseXml($this->getResponse());
		$responseResources = [];
		$hrefs = $responseXmlObj->xpath('//d:href');
		foreach ($hrefs as $href) {
			$hrefParts = \explode("/", (string)$href[0]);
			if (\in_array($user, $hrefParts)) {
				$entry = \urldecode(\end($hrefParts));
				$responseResources[] = $entry;
			} else {
				throw new Error("Expected user: $hrefParts[5] but found: $user");
			}
		}
		return $responseResources;
	}

	/**
	 * parses a PROPFIND response from $this->response into xml
	 * and returns found search results if found else returns false
	 *
	 * @param string|null $user
	 *
	 * @return int
	 */
	public function getNumberOfEntriesInPropfindResponse(
		?string $user = null
	): int {
		$multistatusResults = $this->getMultiStatusResultFromPropfindResult($user);
		return \count($multistatusResults);
	}

	/**
	 * parses a PROPFIND response from $this->response
	 * and returns multistatus data from the response
	 *
	 * @param string|null $user
	 *
	 * @return array
	 */
	public function getMultiStatusResultFromPropfindResult(
		?string $user = null
	): array {
		$responseXmlArray = HttpRequestHelper::parseResponseAsXml($this->response);

		Assert::assertNotEmpty($responseXmlArray, __METHOD__ . ' Response is empty');
		if ($user === null) {
			$user = $this->getCurrentUser();
		}
		Assert::assertIsArray(
			$responseXmlArray,
			__METHOD__ . " response for user $user is not an array"
		);
		Assert::assertArrayHasKey(
			"value",
			$responseXmlArray,
			__METHOD__ . " response for user $user does not have key 'value'"
		);
		$multistatus = $responseXmlArray["value"];
		if ($multistatus == null) {
			$multistatus = [];
		}
		return $multistatus;
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
	 * parses a PROPFIND response from $this->response into xml
	 * and returns found search results if found else returns false
	 *
	 * @param string|null $entryNameToSearch
	 * @param string|null $user
	 * @param string $type
	 * @param string $folderPath
	 * @param string|null $spaceId
	 *
	 * @return string|array|boolean
	 *
	 * string if $entryNameToSearch is given and is found
	 * array if $entryNameToSearch is not given
	 * boolean false if $entryNameToSearch is given and is not found
	 *
	 * @throws GuzzleException
	 */
	public function findEntryFromPropfindResponse(
		?string $entryNameToSearch = null,
		?string $user = null,
		string $type = "files",
		string $folderPath = '',
		?string $spaceId = null
	) {
		$trimmedEntryNameToSearch = '';
		// trim any leading "/" passed by the caller, we can just match the "raw" name
		if ($entryNameToSearch != null) {
			$trimmedEntryNameToSearch = \trim($entryNameToSearch, "/");
		}
		// url encode for spaces and brackets that may appear in the filePath
		$folderPath = $this->escapePath($folderPath);
		// topWebDavPath should be something like '/webdav/' or '/dav/files/{user}/'
		$topWebDavPath = "/" . $this->getFullDavFilesPath($user, $spaceId) . "/" . $folderPath;
		switch ($type) {
			case "files":
				break;
			case "public-files":
				$topWebDavPath = "/" . $this->getPublicLinkDavPath($user) . "/";
				break;
			default:
				throw new Exception("error");
		}

		$multistatusResults = $this->getMultiStatusResultFromPropfindResult($user);

		$results = [];
		foreach ($multistatusResults as $multistatusResult) {
			$entryPath = $multistatusResult['value'][0]['value'];
			$entryName = \str_replace($topWebDavPath, "", $entryPath);
			$entryName = \rawurldecode($entryName);
			$entryName = \trim($entryName, "/");
			if ($trimmedEntryNameToSearch === $entryName) {
				return $multistatusResult;
			}
			$results[] = $entryName;
		}
		if ($entryNameToSearch === null) {
			return $results;
		}
		return false;
	}

	/**
	 * parses a REPORT response from $this->response into xml
	 * and returns found search results if found else returns false
	 *
	 * @param string|null $entryNameToSearch
	 * @param bool|null $searchForHighlightString
	 * @param string|null $spaceId
	 *
	 * @return string|array|boolean
	 *
	 * string if $entryNameToSearch is given and is found
	 * array if $entryNameToSearch is not given
	 * boolean false if $entryNameToSearch is given and is not found
	 *
	 * @throws Exception
	 */
	public function findEntryFromSearchResponse(
		?string $entryNameToSearch = null,
		?bool $searchForHighlightString = false,
		?string $spaceId = null
	) {
		// trim any leading "/" passed by the caller, we can just match the "raw" name
		if ($entryNameToSearch !== null) {
			$entryNameToSearch = \trim($entryNameToSearch, "/");
		}

		$spacesBaseUrl = "/" . WebDavHelper::getDavPath($this->getDavPathVersion(), $spaceId);
		$spacesBaseUrl = \rtrim($spacesBaseUrl, "/") . "/";
		$hrefRegex = \preg_quote($spacesBaseUrl, "/");
		if (\in_array($this->getDavPathVersion(), [WebDavHelper::DAV_VERSION_SPACES, WebDavHelper::DAV_VERSION_NEW])
			&& !GraphHelper::isSpaceId($entryNameToSearch ?? '')
		) {
			$hrefRegex .= "[a-zA-Z0-9-_$!:%]+";
		}
		$hrefRegex = "/^" . $hrefRegex . "/";

		$searchResults = HttpRequestHelper::getResponseXml($this->getResponse())->xpath("//d:multistatus/d:response");

		$results = [];
		foreach ($searchResults as $item) {
			$href = (string)$item->xpath("d:href")[0];
			$shareRootXml = $item->xpath("d:propstat//oc:shareroot");
			$resourcePath = \preg_replace($hrefRegex, "", $href);
			// do not try to parse the resource path
			// if the item to search is space itself
			if (!GraphHelper::isSpaceId($entryNameToSearch ?? '')) {
				$resourcePath = basename($resourcePath);
			}
			$resourcePath = \rawurldecode($resourcePath);
			if ($entryNameToSearch === $resourcePath) {
				// If searching for a single entry,
				// we return a SimpleXmlElement of found item
				return $item;
			}
			if ($searchForHighlightString) {
				// If searching for highlighted string,
				// we return an array of entries with highlighted content as value
				// Example:
				//      [
				//          "<entryName1>" => "<highlighted-content>"
				//          "<entryName2>" => "<highlighted-content>"
				//      ]
				$actualHighlightString =  $item->xpath("d:propstat//oc:highlights");
				$results[$resourcePath] = (string)$actualHighlightString[0];
			} else {
				// If list all the entries i.e. $entryNameToSearch=null,
				// we return an array of entries in the response
				// Example:
				//      ["<entry1>", "<entry2>"]
				$results[] = $resourcePath;
			}
		}
		if ($entryNameToSearch === null) {
			return $results;
		}
		return false;
	}

	/**
	 * Prevent creating two uploads and/or deletes with the same "stime"
	 * That is based on seconds in some implementations.
	 * This prevents duplication of etags or other problems with
	 * trashbin/versions save/restore.
	 *
	 * Set env var UPLOAD_DELETE_WAIT_TIME to 1 to activate a 1-second pause.
	 * By default, there is no pause. That allows testing of implementations
	 * which should be able to cope with multiple upload/delete actions in the
	 * same second.
	 *
	 * @return void
	 */
	public function pauseUploadDelete(): void {
		$time = \time();
		$uploadWaitTime = \getenv("UPLOAD_DELETE_WAIT_TIME");

		$uploadWaitTime = $uploadWaitTime ? (int)$uploadWaitTime : 0;

		if (($this->lastUploadDeleteTime !== null)
			&& ($uploadWaitTime > 0)
			&& (($time - $this->lastUploadDeleteTime) < $uploadWaitTime)
		) {
			\sleep($uploadWaitTime);
		}
	}

	/**
	 * @param string $index
	 * @param string $expectedUsername
	 *
	 * @return void
	 */
	public function checkAuthorOfAVersionOfFile(string $index, string $expectedUsername): void {
		$expectedUserDisplayName = $this->getUserDisplayName($expectedUsername);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$this->getResponse(),
			__METHOD__
		);

		// the username should be in oc:meta-version-edited-by
		$xmlPart = $responseXmlObject->xpath("//oc:meta-version-edited-by");
		$authors = [];
		foreach ($xmlPart as $idx => $author) {
			// The first element is the root path element which is not a version
			// So skipping it
			if ($idx !== 0) {
				$authors[] = $author->__toString();
			}
		}
		if (!isset($authors[$index - 1])) {
			Assert::fail(
				'could not find version with index "' . $index
				. '" for oc:meta-version-edited-by property in response to user "' . $this->responseUser . '"'
			);
		}
		$actualUser = $authors[$index - 1];
		Assert::assertEquals(
			$expectedUsername,
			$actualUser,
			"Expected user of version with index $index in response to user '$this->responseUser'"
			. " was '$expectedUsername', but got '$actualUser'"
		);

		// the user's display name should be in oc:meta-version-edited-by-name
		$xmlPart = $responseXmlObject->xpath("//oc:meta-version-edited-by-name");
		$displaynames = [];
		foreach ($xmlPart as $idx => $displayname) {
			// The first element is the root path element which is not a version
			// So skipping it
			if ($idx !== 0) {
				$displaynames[] = $displayname->__toString();
			}
		}
		if (!isset($displaynames[$index - 1])) {
			Assert::fail(
				'could not find version with index "' . $index
				. '" for oc:meta-version-edited-by-name property in response to user "' . $this->responseUser . '"'
			);
		}
		$actualUserDisplayName = $displaynames[$index - 1];
		Assert::assertEquals(
			$expectedUserDisplayName,
			$actualUserDisplayName,
			"Expected display name of version with index $index in response to user '$this->responseUser'"
			. " was '$expectedUserDisplayName', but got '$actualUserDisplayName'"
		);
	}

	/**
	 * @When user :user downloads the content of GDPR report :pathToFile
	 *
	 * @param string $user
	 * @param string $pathToFile
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userGetsTheContentOfGeneratedJsonReport(string $user, string $pathToFile): void {
		$password = $this->getPasswordForUser($user);
		$response = $this->downloadFileAsUserUsingPassword($user, $pathToFile, $password);
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}
}
