<?php declare(strict_types=1);
/**
 * @author Artur Neumann <artur@jankaritech.com>
 * @copyright Copyright (c) 2018, ownCloud GmbH
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
use GuzzleHttp\Exception\GuzzleException;
use PHPUnit\Framework\Assert;
use Psr\Http\Message\ResponseInterface;
use TestHelpers\HttpRequestHelper;
use TestHelpers\WebDavHelper;
use TestHelpers\BehatHelper;
use TestHelpers\UploadHelper;

require_once 'bootstrap.php';

/**
 * context file for steps that execute actions as "the public".
 */
class PublicWebDavContext implements Context {
	private FeatureContext $featureContext;

	/**
	 * @param string $range ignore if empty
	 * @param string|null $password
	 *
	 * @return ResponseInterface
	 */
	public function downloadPublicFileWithRange(string $range, ?string $password = ""): ResponseInterface {
		// In this case a single file has been shared as a public link.
		// Even if that file is somewhere down inside a folder(s), when
		// accessing it as a public link using the public webDAV API
		// the client needs to provide the public link share token followed
		// by just the name of the file - not the full path.
		$fullPath = (string) $this->featureContext->getLastCreatedPublicShare()->path;
		$fullPathParts = \explode("/", $fullPath);
		$path = \end($fullPathParts);

		return $this->downloadFileFromPublicFolder(
			$path,
			$password,
			$range
		);
	}

	/**
	 * @When /^the public downloads the last public link shared file using the public WebDAV API$/
	 * @When /^the public tries to download the last public link shared file using the public WebDAV API$/
	 *
	 * @return void
	 */
	public function downloadPublicFile(): void {
		$response = $this->downloadPublicFileWithRange("");
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When /^the public downloads the last public link shared file with password "([^"]*)" using the public WebDAV API$/
	 * @When /^the public tries to download the last public link shared file with password "([^"]*)" using the public WebDAV API$/
	 *
	 * @param string $password
	 *
	 * @return void
	 */
	public function downloadPublicFileWithPassword(string $password): void {
		$response = $this->downloadPublicFileWithRange("", $password);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When /^user "([^"]*)" tries to download file "([^"]*)" from the last public link using own basic auth and public WebDAV API$/
	 *
	 * @param string $user
	 * @param string $path
	 *
	 * @return void
	 */
	public function userTriesToDownloadFileFromPublicLinkUsingBasicAuthAndPublicWebdav(
		string $user,
		string $path
	): void {
		$response = $this->downloadFromPublicLinkAsUser($path, $user);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When /^the public deletes (?:file|folder|entry) "([^"]*)" from the last public link share using the public WebDAV API$/
	 *
	 * @param string $fileName
	 *
	 * @return void
	 */
	public function thePublicDeletesFileFolderFromTheLastPublicLinkShareUsingThePublicWebdavApi(
		string $fileName
	): void {
		$response = $this->deleteFileFromPublicShare(
			$fileName
		);
		$this->featureContext->setResponse($response);
		$this->featureContext->pushToLastStatusCodesArrays();
	}

	/**
	 * @param string $fileName
	 * @param string $password
	 *
	 * @return ResponseInterface
	 */
	public function deleteFileFromPublicShare(string $fileName, string $password = ""): ResponseInterface {
		$token = ($this->featureContext->isUsingSharingNG())
		? $this->featureContext->shareNgGetLastCreatedLinkShareToken()
		: $this->featureContext->getLastCreatedPublicShareToken();
		$davPath = WebDavHelper::getDavPath(
			WebDavHelper::DAV_VERSION_NEW,
			$token,
			"public-files"
		);
		$password = $this->featureContext->getActualPassword($password);
		$fullUrl = $this->featureContext->getBaseUrl() . "/$davPath/$fileName";
		$userName = $this->getUsernameForPublicWebdavApi($password);
		$headers = [
			'X-Requested-With' => 'XMLHttpRequest'
		];
		return HttpRequestHelper::delete(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			$userName,
			$password,
			$headers
		);
	}

	/**
	 * @Given /^the public has deleted (?:file|folder|entry) "([^"]*)" from the last link share with password "([^"]*)" using the public WebDAV API$/
	 *
	 * @param string $file
	 * @param string $password
	 *
	 * @return void
	 */
	public function thePublicHasDeletedFileFromTheLastLinkShareWithPasswordUsingPublicWebdavApi(
		string $file,
		string $password
	): void {
		$response = $this->deleteFileFromPublicShare($file, $password);
		$this->featureContext->theHTTPStatusCodeShouldBe([201, 204], "", $response);
	}

	/**
	 * @When /^the public deletes (?:file|folder|entry) "([^"]*)" from the last link share with password "([^"]*)" using the public WebDAV API$/
	 *
	 * @param string $file
	 * @param string $password
	 *
	 * @return void
	 */
	public function thePublicDeletesFileFromTheLastLinkShareWithPasswordUsingPublicWebdavApi(
		string $file,
		string $password
	): void {
		$this->featureContext->setResponse(
			$this->deleteFileFromPublicShare($file, $password)
		);
		$this->featureContext->pushToLastStatusCodesArrays();
	}

	/**
	 * @param string $fileName
	 * @param string $toFileName
	 * @param string|null $password
	 *
	 * @return ResponseInterface
	 */
	public function renameFileFromPublicShare(
		string $fileName,
		string $toFileName,
		?string $password = ""
	): ResponseInterface {
		$token = ($this->featureContext->isUsingSharingNG())
		? $this->featureContext->shareNgGetLastCreatedLinkShareToken()
		: $this->featureContext->getLastCreatedPublicShareToken();
		$davPath = WebDavHelper::getDavPath(
			WebDavHelper::DAV_VERSION_NEW,
			$token,
			"public-files"
		);
		$fullUrl = $this->featureContext->getBaseUrl() . "/$davPath/$fileName";
		$password = $this->featureContext->getActualPassword($password);
		$destination = $this->featureContext->getBaseUrl() . "/$davPath/$toFileName";
		$userName = $this->getUsernameForPublicWebdavApi($password);
		$headers = [
			'X-Requested-With' => 'XMLHttpRequest',
			'Destination' => $destination
		];
		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			"MOVE",
			$userName,
			$password,
			$headers
		);
	}

	/**
	 * @When /^the public renames (?:file|folder|entry) "([^"]*)" to "([^"]*)" from the last public link share using the password "([^"]*)" and the public WebDAV API$/
	 *
	 * @param string $fileName
	 * @param string $toName
	 * @param string $password
	 *
	 * @return void
	 */
	public function thePublicRenamesFileFromTheLastPublicShareUsingThePasswordPasswordAndOldPublicWebdavApi(
		string $fileName,
		string $toName,
		string $password
	): void {
		$this->featureContext->setResponse(
			$this->renameFileFromPublicShare($fileName, $toName, $password)
		);
		$this->featureContext->pushToLastStatusCodesArrays();
	}

	/**
	 * @When /^the public downloads file "([^"]*)" from inside the last public link shared folder with password "([^"]*)" using the public WebDAV API$/
	 * @When /^the public tries to download file "([^"]*)" from inside the last public link shared folder with password "([^"]*)" using the public WebDAV API$/
	 *
	 * @param string $path
	 * @param string $password
	 *
	 * @return void
	 */
	public function publicDownloadsFileFromInsideLastPublicSharedFolderWithPassword(
		string $path,
		string $password = ""
	): void {
		$response = $this->downloadFileFromPublicFolder(
			$path,
			$password,
			"",
			$this->featureContext->isUsingSharingNG()
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @param string $path
	 * @param string $user
	 *
	 * @return ResponseInterface
	 */
	public function downloadFromPublicLinkAsUser(string $path, string $user): ResponseInterface {
		$path = \ltrim($path, "/");
		$token = ($this->featureContext->isUsingSharingNG())
		? $this->featureContext->shareNgGetLastCreatedLinkShareToken()
		: $this->featureContext->getLastCreatedPublicShareToken();

		$davPath = WebDavHelper::getDavPath(
			$this->featureContext->getDavPathVersion(),
			$token,
			"public-files"
		);

		$username = $this->featureContext->getActualUsername($user);
		$password = $this->featureContext->getPasswordForUser($user);
		$fullUrl = $this->featureContext->getBaseUrl() . "/$davPath/$path";

		return HttpRequestHelper::get(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			$username,
			$password
		);
	}
	/**
	 * @param string $path
	 * @param string $password
	 * @param string $range ignored when empty
	 * @param bool $shareNg
	 *
	 * @return ResponseInterface
	 */
	public function downloadFileFromPublicFolder(
		string $path,
		string $password,
		string $range,
		bool $shareNg = false
	): ResponseInterface {
		$path = \ltrim($path, "/");
		$password = $this->featureContext->getActualPassword($password);
		if ($shareNg) {
			$token = $this->featureContext->shareNgGetLastCreatedLinkShareToken();
		} else {
			$token = $this->featureContext->getLastCreatedPublicShareToken();
		}
		$davPath = WebDavHelper::getDavPath(
			WebDavHelper::DAV_VERSION_NEW,
			$token,
			"public-files"
		);
		$fullUrl = $this->featureContext->getBaseUrl() . "/$davPath/$path";
		$userName = $this->getUsernameForPublicWebdavApi($password);

		$headers = [
			'X-Requested-With' => 'XMLHttpRequest'
		];
		if ($range !== "") {
			$headers['Range'] = $range;
		}
		return HttpRequestHelper::get(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			$userName,
			$password,
			$headers
		);
	}

	/**
	 * @param string $source target file name
	 *
	 * @return ResponseInterface
	 */
	public function publicUploadFile(string $source): ResponseInterface {
		return $this->publicUploadContent(
			\basename($source),
			'',
			\file_get_contents($source)
		);
	}

	/**
	 *
	 * @param string $baseUrl URL of OpenCloud
	 * @param string $source
	 * @param string $destination
	 *
	 * @return ResponseInterface
	 */
	public function publiclyCopyingFile(
		string $baseUrl,
		string $source,
		string $destination
	): ResponseInterface {
		$fullSourceUrl = "$baseUrl/$source";
		$fullDestUrl = WebDavHelper::sanitizeUrl(
			"$baseUrl/$destination"
		);

		$headers["Destination"] = $fullDestUrl;
		return HttpRequestHelper::sendRequest(
			$fullSourceUrl,
			$this->featureContext->getStepLineRef(),
			"COPY",
			null,
			null,
			$headers
		);
	}

	/**
	 * @When /^the public copies (?:file|folder) "([^"]*)" to "([^"]*)" using the public WebDAV API$/
	 *
	 * @param string $source
	 * @param string $destination
	 *
	 * @return void
	 */
	public function thePublicCopiesFileUsingTheWebDAVApi(string $source, string $destination): void {
		$token = ($this->featureContext->isUsingSharingNG())
		? $this->featureContext->shareNgGetLastCreatedLinkShareToken()
		: $this->featureContext->getLastCreatedPublicShareToken();
		$davPath = WebDavHelper::getDavPath(
			WebDavHelper::DAV_VERSION_NEW,
			$token,
			"public-files"
		);
		$baseUrl = $this->featureContext->getLocalBaseUrl() . '/' . $davPath;

		$response = $this->publiclyCopyingFile(
			$baseUrl,
			$source,
			$destination
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @param string $filename target file name
	 * @param string $password
	 * @param string $body content to upload
	 *
	 * @return ResponseInterface
	 */
	public function publiclyUploadingContentWithPassword(
		string $filename,
		string $password = '',
		string $body = 'test',
	): ResponseInterface {
		return $this->publicUploadContent(
			$filename,
			$password,
			$body,
			true,
			[],
		);
	}

	/**
	 * @Given /^the public has uploaded file "([^"]*)" with content "([^"]*)" and password "([^"]*)" to the last link share using the public WebDAV API$/
	 *
	 * @param string $filename
	 * @param string $content
	 * @param string $password
	 *
	 * @return void
	 */
	public function thePublicHasUploadedFileWithContentAndPasswordToLastLinkShareUsingPublicWebdavApi(
		string $filename,
		string $content = 'test',
		string $password = ''
	): void {
		$response = $this->publiclyUploadingContentWithPassword(
			$filename,
			$password,
			$content,
		);
		$this->featureContext->theHTTPStatusCodeShouldBe([201, 204], "", $response);
	}

	/**
	 * @When /^the public uploads file "([^"]*)" with password "([^"]*)" and content "([^"]*)" using the public WebDAV API$/
	 *
	 * @param string $filename target file name
	 * @param string $password
	 * @param string $body content to upload
	 *
	 * @return void
	 */
	public function thePublicUploadsFileWithPasswordAndContentUsingPublicWebDAVApi(
		string $filename,
		string $password,
		string $body,
	): void {
		$response = $this->publiclyUploadingContentWithPassword(
			$filename,
			$password,
			$body
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When the public overwrites file :filename with content :body using the public WebDAV API
	 *
	 * @param string $filename target file name
	 * @param string $body content to upload
	 *
	 * @return void
	 */
	public function thePublicOverwritesFileWithContentUsingWebDavApi(string $filename, string $body): void {
		$response = $this->publicUploadContent($filename, '', $body);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When /^the public uploads file "([^"]*)" with content "([^"]*)" using the public WebDAV API$/
	 *
	 * @param string $filename target file name
	 * @param string $body content to upload
	 *
	 * @return void
	 */
	public function thePublicUploadsFileWithContentUsingThePublicWebDavApi(
		string $filename,
		string $body = 'test'
	): void {
		$response = $this->publicUploadContent($filename, '', $body);
		$this->featureContext->setResponse($response);
		$this->featureContext->pushToLastStatusCodesArrays();
	}

	/**
	 * @Given the public has uploaded file :filename with content :body
	 *
	 * @param string $filename target file name
	 * @param string $body content to upload
	 *
	 * @return void
	 */
	public function thePublicHasUploadedFileWithContent(string $filename, string $body): void {
		$response = $this->publicUploadContent($filename, '', $body);
		$this->featureContext->theHTTPStatusCodeShouldBe([201, 204], "", $response);
	}

	/**
	 * @Then /^the public should be able to download the last publicly shared file using the public WebDAV API with password "([^"]*)" and the content should be "([^"]*)"$/
	 *
	 * @param string $password
	 * @param string $expectedContent
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkLastPublicSharedFileWithPasswordDownload(
		string $password,
		string $expectedContent
	): void {
		$response = $this->downloadPublicFileWithRange(
			"",
			$password
		);

		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);

		$this->featureContext->checkDownloadedContentMatches(
			$expectedContent,
			"Checking the content of the last public shared file after downloading with the public WebDAV API",
			$response
		);
	}

	/**
	 * @Then /^the public should not be able to download file "([^"]*)" from inside the last public link shared folder using the public WebDAV API without a password$/
	 * @Then /^the public download of file "([^"]*)" from inside the last public link shared folder using the public WebDAV API should fail with HTTP status code "([^"]*)"$/
	 *
	 * @param string $path
	 * @param string $expectedHttpCode
	 *
	 * @return void
	 */
	public function shouldNotBeAbleToDownloadFileInsidePublicSharedFolder(
		string $path,
		string $expectedHttpCode = "401"
	): void {
		$response = $this->downloadFileFromPublicFolder(
			$path,
			"",
			"",
			$this->featureContext->isUsingSharingNG()
		);
		$this->featureContext->theHTTPStatusCodeShouldBe($expectedHttpCode, "", $response);
	}

	/**
	 * @Then /^the public should be able to download file "([^"]*)" from inside the last public link shared folder using the public WebDAV API with password "([^"]*)"$/
	 *
	 * @param string $path
	 * @param string $password
	 *
	 * @return void
	 */
	public function shouldBeAbleToDownloadFileInsidePublicSharedFolderWithPassword(
		string $path,
		string $password
	): void {
		$response = $this->downloadFileFromPublicFolder(
			$path,
			$password,
			"",
			$this->featureContext->isUsingSharingNG()
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
	}

	/**
	 * @Then /^the public should be able to download file "([^"]*)" from inside the last public link shared folder using the public WebDAV API with password "([^"]*)" and the content should be "([^"]*)"$/
	 *
	 * @param string $path
	 * @param string $password
	 * @param string $content
	 *
	 * @return void
	 * @throws Exception
	 */
	public function shouldBeAbleToDownloadFileInsidePublicSharedFolderWithPasswordAndContentShouldBe(
		string $path,
		string $password,
		string $content
	): void {
		$response = $this->downloadFileFromPublicFolder(
			$path,
			$password,
			"",
		);
		$this->featureContext->checkDownloadedContentMatches($content, "", $response);

		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
	}

	/**
	 * @Then /^the public should be able to download file "([^"]*)" from the last link share with password "([^"]*)" and the content should be "([^"]*)"$/
	 *
	 * @param string $path
	 * @param string $password
	 * @param string $content
	 *
	 * @return void
	 * @throws Exception
	 */
	public function shouldBeAbleToDownloadFileInsidePublicSharedFolderWithPasswordForSharingNGAndContentShouldBe(
		string $path,
		string $password,
		string $content
	): void {
		$response = $this->downloadFileFromPublicFolder(
			$path,
			$password,
			"",
			true
		);

		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
		$this->featureContext->checkDownloadedContentMatches($content, "", $response);
	}

	/**
	 * @Then /^the public should not be able to download file "([^"]*)" from inside the last public link shared folder using the public WebDAV API with password "([^"]*)"$/
	 * @Then /^the public download of file "([^"]*)" from inside the last public link shared folder using the public WebDAV API with password "([^"]*)" should fail with HTTP status code "([^"]*)"$/
	 *
	 * @param string $path
	 * @param string $password
	 * @param string $expectedHttpCode
	 *
	 * @return void
	 */
	public function shouldNotBeAbleToDownloadFileInsidePublicSharedFolderWithPassword(
		string $path,
		string $password,
		string $expectedHttpCode = "401"
	): void {
		$response = $this->downloadFileFromPublicFolder(
			$path,
			$password,
			"",
			$this->featureContext->isUsingSharingNG()
		);
		$this->featureContext->theHTTPStatusCodeShouldBe($expectedHttpCode, "", $response);
	}

	/**
	 * @Then /^the public download of file "([^"]*)" from the last link share with password "([^"]*)" should fail with HTTP status code "([^"]*)" using shareNg$/
	 *
	 * @param string $path
	 * @param string $password
	 * @param string $expectedHttpCode
	 *
	 * @return void
	 */
	public function shouldNotBeAbleToDownloadFileWithPasswordForShareNg(
		string $path,
		string $password,
		string $expectedHttpCode = "401"
	): void {
		$this->tryingToDownloadUsingWebDAVAPI(
			$path,
			"new",
			$password,
			$expectedHttpCode,
			true
		);
	}

	/**
	 * @param string $path
	 * @param string $password
	 * @param string $range
	 * @param string $expectedHttpCode
	 * @param boolean $shareNg
	 *
	 * @return void
	 */
	public function tryingToDownloadUsingWebDAVAPI(
		string $path,
		string $password,
		string $range = "",
		string $expectedHttpCode = "401",
		bool $shareNg = false
	): void {
		$response = $this->downloadFileFromPublicFolder(
			$path,
			$password,
			$range,
			$shareNg
		);
		$this->featureContext->theHTTPStatusCodeShouldBe($expectedHttpCode, "", $response);
	}

	/**
	 * @Then /^the public upload to the last publicly shared file using the public WebDAV API with password "([^"]*)" should (?:fail|pass) with HTTP status code "([^"]*)"$/
	 *
	 * @param string $password
	 * @param string $expectedHttpCode
	 *
	 * @return void
	 * @throws Exception
	 */
	public function publiclyUploadingShouldToSharedFileShouldFail(
		string $password,
		string $expectedHttpCode
	): void {
		$filename = (string)$this->featureContext->getLastCreatedPublicShare()->file_target;

		$response = $this->publicUploadContent(
			$filename,
			$password
		);

		$this->featureContext->theHTTPStatusCodeShouldBe($expectedHttpCode, "", $response);
	}

	/**
	 * @Then /^the public upload to the last publicly shared folder using the public WebDAV API with password "([^"]*)" should fail with HTTP status code "([^"]*)"$/
	 *
	 * @param string $password
	 * @param string|null $expectedHttpCode
	 *
	 * @return void
	 */
	public function publiclyUploadingWithPasswordShouldNotWork(
		string $password,
		string $expectedHttpCode = null
	): void {
		$response = $this->publicUploadContent(
			'whateverfilefortesting.txt',
			$password
		);
		Assert::assertGreaterThanOrEqual(
			$expectedHttpCode,
			$response->getStatusCode(),
			"upload should have failed but passed with code " . $response->getStatusCode()
		);
	}

	/**
	 * @Then /^the public should be able to upload file "([^"]*)" into the last public link shared folder using the public WebDAV API with password "([^"]*)"$/
	 *
	 * @param string $filename
	 * @param string $password
	 *
	 * @return void
	 */
	public function publiclyUploadingIntoFolderWithPasswordShouldWork(
		string $filename,
		string $password
	): void {
		$response = $this->publicUploadContent(
			$filename,
			$password
		);

		$this->featureContext->theHTTPStatusCodeShouldBe([201, 204], "", $response);
	}

	/**
	 * @Then /^uploading a file with password "([^"]*)" should work using the public WebDAV API$/
	 *
	 * @param string $password
	 *
	 * @return void
	 * @throws Exception
	 */
	public function publiclyUploadingShouldWork(string $password): void {
		$path = "whateverfilefortesting-publicWebDAVAPI.txt";
		$content = "test";

		$response = $this->publicUploadContent(
			$path,
			$password,
			$content,
		);

		Assert::assertTrue(
			($response->getStatusCode() === 201),
			"upload should have passed but failed with code " .
			$response->getStatusCode()
		);
		$response = $this->downloadFileFromPublicFolder(
			$path,
			$password,
			""
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
		$this->featureContext->checkDownloadedContentMatches($content, "", $response);
	}

	/**
	 * @Then /^uploading content to a public link shared file with password "([^"]*)" should (not|)\s?work using the public WebDAV API$/
	 *
	 * @param string $password
	 * @param string $shouldOrNot (not|)
	 *
	 * @return void
	 * @throws Exception
	 */
	public function publiclyUploadingToPublicLinkSharedFileShouldWork(
		string $password,
		string $shouldOrNot,
	): void {
		$content = "test";
		$should = ($shouldOrNot !== "not");
		$path = (string) $this->featureContext->getLastCreatedPublicShare()->path;

		$response = $this->publicUploadContent(
			$path,
			$password,
			$content
		);
		if ($should) {
			Assert::assertTrue(
				($response->getStatusCode() == 204),
				"upload should have passed but failed with code " .
				$response->getStatusCode()
			);

			$response = $this->downloadPublicFileWithRange(
				"",
				$password
			);

			$this->featureContext->checkDownloadedContentMatches(
				$content,
				"Checking the content of the last public shared file after downloading with the public WebDAV API",
				$response
			);
		} else {
			$expectedCode = 403;
			Assert::assertTrue(
				($response->getStatusCode() == $expectedCode),
				"upload should have failed with HTTP status $expectedCode but passed with code " .
				$response->getStatusCode()
			);
		}
	}

	/**
	 * @When the public uploads file :source to :destination inside last link shared folder using the public WebDAV API
	 *
	 * @param string $source
	 * @param string $destination
	 *
	 * @return void
	 */
	public function thePublicUploadsFileToInsideLastLinkSharedFolderUsingThePublicWebdavApi(
		string $source,
		string $destination,
	): void {
		$content = \file_get_contents(
			UploadHelper::getAcceptanceTestsDir() . $source
		);
		$response = $this->publicUploadContent(
			$destination,
			'',
			$content
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @When the public uploads file :source to :destination inside last link shared folder with password :password using the public WebDAV API
	 *
	 * @param string $source
	 * @param string $destination
	 * @param string $password
	 *
	 * @return void
	 */
	public function thePublicUploadsFileToInsideLastLinkSharedFolderWithPasswordUsingThePublicWebdavApi(
		string $source,
		string $destination,
		string $password
	): void {
		$content = \file_get_contents(
			UploadHelper::getAcceptanceTestsDir() . $source
		);
		$response = $this->publicUploadContent(
			$destination,
			$password,
			$content
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @When the public uploads file :fileName to the last public link shared folder with password :password with mtime :mtime using the public WebDAV API
	 *
	 * @param string $fileName
	 * @param string $password
	 * @param string $mtime
	 *
	 * @return void
	 * @throws Exception
	 */
	public function thePublicUploadsFileToLastSharedFolderWithMtimeUsingTheWebdavApi(
		string $fileName,
		string $password,
		string $mtime
	): void {
		$mtime = new DateTime($mtime);
		$mtime = $mtime->format('U');

		$response = $this->publicUploadContent(
			$fileName,
			$password,
			'test',
			false,
			["X-OC-Mtime" => $mtime]
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @param string $destination
	 * @param string $password
	 *
	 * @return ResponseInterface
	 */
	public function publicCreatesFolderUsingPassword(
		string $destination,
		string $password
	): ResponseInterface {
		$token = ($this->featureContext->isUsingSharingNG())
		? $this->featureContext->shareNgGetLastCreatedLinkShareToken()
		: $this->featureContext->getLastCreatedPublicShareToken();
		$davPath = WebDavHelper::getDavPath(
			$this->featureContext->getDavPathVersion(),
			$token,
			"public-files"
		);
		$url = $this->featureContext->getBaseUrl() . "/$davPath/";
		$password = $this->featureContext->getActualPassword($password);
		$userName = $this->getUsernameForPublicWebdavApi($password);
		$foldername = \implode(
			'/',
			\array_map('rawurlencode', \explode('/', $destination))
		);
		$url .= \ltrim($foldername, '/');

		return HttpRequestHelper::sendRequest(
			$url,
			$this->featureContext->getStepLineRef(),
			'MKCOL',
			$userName,
			$password
		);
	}

	/**
	 * @When the public creates folder :destination using the public WebDAV API
	 *
	 * @param String $destination
	 *
	 * @return void
	 */
	public function publicCreatesFolder(string $destination): void {
		$this->featureContext->setResponse($this->publicCreatesFolderUsingPassword($destination, ''));
	}

	/**
	 * @Then /^the public should be able to create folder "([^"]*)" in the last public link shared folder using the new public WebDAV API with password "([^"]*)"$/
	 *
	 * @param string $foldername
	 * @param string $password
	 *
	 * @return void
	 */
	public function publicShouldBeAbleToCreateFolderWithPassword(
		string $foldername,
		string $password
	): void {
		$response = $this->publicCreatesFolderUsingPassword($foldername, $password);
		$this->featureContext->theHTTPStatusCodeShouldBe(201, "", $response);
	}

	/**
	 * @Then /^the public creation of folder "([^"]*)" in the last public link shared folder using the new public WebDAV API with password "([^"]*)" should fail with HTTP status code "([^"]*)"$/
	 *
	 * @param string $foldername
	 * @param string $password
	 * @param string $expectedHttpCode
	 *
	 * @return void
	 */
	public function publicCreationOfFolderWithPasswordShouldFail(
		string $foldername,
		string $password,
		string $expectedHttpCode
	): void {
		$response = $this->publicCreatesFolderUsingPassword($foldername, $password);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHttpCode,
			"creation of $foldername in the last publicly shared folder should have failed with code " .
			$expectedHttpCode,
			$response
		);
	}

	/**
	 * @Then the mtime of file :fileName in the last shared public link using the WebDAV API should be :mtime
	 *
	 * @param string $fileName
	 * @param string $mtime
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theMtimeOfFileInTheLastSharedPublicLinkUsingTheWebdavApiShouldBe(
		string $fileName,
		string $mtime
	): void {
		$token = ($this->featureContext->isUsingSharingNG())
		? $this->featureContext->shareNgGetLastCreatedLinkShareToken()
		: $this->featureContext->getLastCreatedPublicShareToken();
		$baseUrl = $this->featureContext->getBaseUrl();
		$mtime = \explode(" ", $mtime);
		\array_pop($mtime);
		$mtime = \implode(" ", $mtime);
		Assert::assertStringContainsString(
			$mtime,
			WebDavHelper::getMtimeOfFileInPublicLinkShare(
				$baseUrl,
				$fileName,
				$token,
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 * @Then the mtime of file :fileName in the last shared public link using the WebDAV API should not be :mtime
	 *
	 * @param string $fileName
	 * @param string $mtime
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theMtimeOfFileInTheLastSharedPublicLinkUsingTheWebdavApiShouldNotBe(
		string $fileName,
		string $mtime
	): void {
		$token = $this->featureContext->getLastCreatedPublicShareToken();
		$baseUrl = $this->featureContext->getBaseUrl();
		Assert::assertNotEquals(
			$mtime,
			WebDavHelper::getMtimeOfFileInPublicLinkShare(
				$baseUrl,
				$fileName,
				$token,
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 * Uploads a file through the public WebDAV API and sets the response in FeatureContext
	 *
	 * @param string $filename
	 * @param string|null $password
	 * @param string $body
	 * @param bool $autoRename
	 * @param array $additionalHeaders
	 *
	 * @return ResponseInterface|null
	 */
	public function publicUploadContent(
		string $filename,
		?string $password = '',
		string $body = 'test',
		bool $autoRename = false,
		array $additionalHeaders = [],
	): ?ResponseInterface {
		$password = $this->featureContext->getActualPassword($password);
		if ($this->featureContext->isUsingSharingNG()) {
			$token = $this->featureContext->shareNgGetLastCreatedLinkShareToken();
		} else {
			$token = $this->featureContext->getLastCreatedPublicShareToken();
		}
		$davPath = WebDavHelper::getDavPath(
			WebDavHelper::DAV_VERSION_NEW,
			$token,
			"public-files"
		);
		$userName = $this->getUsernameForPublicWebdavApi($password);

		$filename = \implode(
			'/',
			\array_map('rawurlencode', \explode('/', $filename))
		);
		$encodedFilePath = \ltrim($filename, '/');
		$url = $this->featureContext->getBaseUrl() . "/$davPath/$encodedFilePath";
		// Trim any "/" from the end. For example, if we are putting content to a
		// single file that has been shared with a link, then the URL should end
		// with the link token and no "/" at the end.
		$url = \rtrim($url, "/");
		$headers = ['X-Requested-With' => 'XMLHttpRequest'];

		if ($autoRename) {
			$headers['OC-Autorename'] = 1;
		}
		$headers = \array_merge($headers, $additionalHeaders);
		return HttpRequestHelper::put(
			$url,
			$this->featureContext->getStepLineRef(),
			$userName,
			$password,
			$headers,
			$body
		);
	}

	/**
	 * @param string $password
	 *
	 * @return string|null
	 */
	private function getUsernameForPublicWebdavApi(
		string $password
	): ?string {
		if ($password !== '') {
			$userName = 'public';
		} else {
			$userName = null;
		}
		return $userName;
	}

	/**
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
	 * @When /^the public sends "([^"]*)" request to the last public link share using the public WebDAV API(?: with password "([^"]*)")?$/
	 *
	 * @param string $method
	 * @param string|null $password
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function publicSendsRequestToLastPublicShare(string $method, ?string $password = ''): void {
		if ($method === "PROPFIND") {
			$body = '<?xml version="1.0"?>
			<d:propfind xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
				<d:prop>
					<d:resourcetype/>
					<oc:public-link-item-type/>
					<oc:public-link-permission/>
					<oc:public-link-expiration/>
					<oc:public-link-share-datetime/>
					<oc:public-link-share-owner/>
				</d:prop>
			</d:propfind>';
		} else {
			$body = null;
		}
		$token = ($this->featureContext->isUsingSharingNG())
		? $this->featureContext->shareNgGetLastCreatedLinkShareToken()
		: $this->featureContext->getLastCreatedPublicShareToken();
		$davPath = WebDavHelper::getDavPath(
			WebDavHelper::DAV_VERSION_NEW,
			$token,
			"public-files"
		);
		$password = $this->featureContext->getActualPassword($password);
		$username = $this->getUsernameForPublicWebdavApi($password);
		$fullUrl = $this->featureContext->getBaseUrl() . "/$davPath";
		$response = HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			$method,
			$username,
			$password,
			null,
			$body
		);
		$this->featureContext->setResponse($response);
	}
}
