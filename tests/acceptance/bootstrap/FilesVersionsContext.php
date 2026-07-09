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
use Behat\Gherkin\Node\TableNode;
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
 * Steps that relate to files_versions app
 */
class FilesVersionsContext implements Context {
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
		// Get all the contexts you need in this context
		$this->featureContext = BehatHelper::getContext($scope, $environment, 'FeatureContext');
		$this->spacesContext = BehatHelper::getContext($scope, $environment, 'SpacesContext');
	}

	/**
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $fileOwner
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :user tries to get versions of file :file from :fileOwner')]
	public function userTriesToGetFileVersions(string $user, string $file, string $fileOwner): void {
		$this->featureContext->setResponse($this->getFileVersions($user, $file, $fileOwner));
	}

	/**
	 *
	 * @param string $user
	 * @param string $file
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :user gets the number of versions of file :file')]
	public function userGetsFileVersions(string $user, string $file): void {
		$this->featureContext->setResponse($this->getFileVersions($user, $file));
	}

	/**
	 *
	 * @param string $file
	 * @param string $password
	 * @param string $fileId
	 *
	 * @return void
	 */
	#[When('the public tries to get the number of versions of file :file with password :password using file-id :endpoint')]
	public function thePublicTriesToGetTheNumberOfVersionsOfFileWithPasswordUsingFileId(
		string $file,
		string $password,
		string $fileId
	): void {
		$password = $this->featureContext->getActualPassword($password);
		$this->featureContext->setResponse(
			$this->featureContext->makeDavRequest(
				"public",
				"PROPFIND",
				$fileId,
				null,
				null,
				null,
				"versions",
				$this->featureContext->getDavPathVersion(),
				false,
				$password
			)
		);
	}

	/**
	 * @param string $user
	 * @param string $file
	 * @param string|null $fileOwner
	 * @param string|null $spaceId
	 *
	 * @return ResponseInterface
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function getFileVersions(
		string $user,
		string $file,
		?string $fileOwner = null,
		?string $spaceId = null,
	): ResponseInterface {
		$user = $this->featureContext->getActualUsername($user);
		$fileOwner = $fileOwner ? $this->featureContext->getActualUsername($fileOwner) : $user;
		$fileId = $this->featureContext->getFileIdForPath($fileOwner, $file, $spaceId);
		Assert::assertNotNull(
			$fileId,
			__METHOD__
			. " fileid of file $file user $fileOwner not found (the file may not exist)"
		);
		return $this->featureContext->makeDavRequest(
			$user,
			"PROPFIND",
			$fileId,
			null,
			null,
			$spaceId,
			"versions"
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $fileId
	 *
	 * @return void
	 */
	#[When('user :user gets the number of versions of file :resource using file-id :fileId')]
	#[When('user :user tries to get the number of versions of file :resource using file-id :fileId')]
	public function userGetsTheNumberOfVersionsOfFileOfTheSpace(string $user, string $fileId): void {
		$this->featureContext->setResponse(
			$this->featureContext->makeDavRequest(
				$user,
				"PROPFIND",
				$fileId,
				null,
				null,
				null,
				"versions"
			)
		);
	}

	/**
	 * @param string $user
	 * @param string $file
	 *
	 * @return ResponseInterface
	 */
	public function getFileVersionMetadata(string $user, string $file): ResponseInterface {
		$user = $this->featureContext->getActualUsername($user);
		$fileId = $this->featureContext->getFileIdForPath($user, $file);
		Assert::assertNotNull(
			$fileId,
			__METHOD__
			. " fileid of file $file user $user not found (the file may not exist)"
		);
		$body = '<?xml version="1.0"?>
                <d:propfind  xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
                  <d:prop>
                    <oc:meta-version-edited-by />
                    <oc:meta-version-edited-by-name />
                  </d:prop>
                </d:propfind>';
		return $this->featureContext->makeDavRequest(
			$user,
			"PROPFIND",
			$fileId,
			null,
			$body,
			null,
			"versions",
		);
	}

	/**
	 * @param string $user
	 * @param int $versionIndex
	 * @param string $path
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function restoreVersionIndexOfFile(string $user, int $versionIndex, string $path): ResponseInterface {
		$user = $this->featureContext->getActualUsername($user);
		$fileId = $this->featureContext->getFileIdForPath($user, $path);
		Assert::assertNotNull(
			$fileId,
			__METHOD__
			. " fileid of file $path user $user not found (the file may not exist)"
		);
		$response = $this->listVersionFolder($user, $fileId, 1);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$response,
			__METHOD__
		);
		$xmlPart = $responseXmlObject->xpath("//d:response/d:href");
		//restoring the version only works with DAV path v2
		$destinationUrl = $this->featureContext->getBaseUrl() . "/" .
			WebDavHelper::getDavPath(WebDavHelper::DAV_VERSION_NEW, $user) . \trim($path, "/");
		$fullUrl = $this->featureContext->getBaseUrlWithoutPath() .
			$xmlPart[$versionIndex];
		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			'COPY',
			$user,
			$this->featureContext->getPasswordForUser($user),
			['Destination' => $destinationUrl]
		);
	}

	/**
	 *
	 * @param string $user
	 * @param int $versionIndex
	 * @param string $path
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Given('user :user has restored version index :versionIndex of file :path')]
	public function userHasRestoredVersionIndexOfFile(string $user, int $versionIndex, string $path): void {
		$response = $this->restoreVersionIndexOfFile($user, $versionIndex, $path);
		$this->featureContext->theHTTPStatusCodeShouldBe(204, "", $response);
	}

	/**
	 *
	 * @param string $user
	 * @param int $versionIndex
	 * @param string $path
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :user restores version index :versionIndex of file :path using the WebDAV API')]
	public function userRestoresVersionIndexOfFile(string $user, int $versionIndex, string $path): void {
		$response = $this->restoreVersionIndexOfFile($user, $versionIndex, $path);
		$this->featureContext->setResponse($response, $user);
	}

	/**
	 * assert file versions count
	 *
	 * @param string $user
	 * @param string $fileId
	 * @param int $expectedCount
	 *
	 * @return void
	 * @throws Exception
	 */
	public function assertFileVersionsCount(string $user, string $fileId, int $expectedCount): void {
		$response = $this->listVersionFolder($user, $fileId, 1);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$response,
			__METHOD__
		);
		$actualCount = \count($responseXmlObject->xpath("//d:prop/d:getetag")) - 1;
		if ($actualCount === -1) {
			$actualCount = 0;
		}
		Assert::assertEquals(
			$expectedCount,
			$actualCount,
			"Expected $expectedCount versions but found $actualCount in \n" . $responseXmlObject->asXML()
		);
	}

	/**
	 *
	 * @param string $path
	 * @param string $user
	 * @param int $count
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the version folder of file :path for user :user should contain :count element(s)')]
	public function theVersionFolderOfFileShouldContainElements(
		string $path,
		string $user,
		int $count
	): void {
		$user = $this->featureContext->getActualUsername($user);
		$fileId = $this->featureContext->getFileIdForPath($user, $path);
		Assert::assertNotNull(
			$fileId,
			__METHOD__
			. ". file '$path' for user '$user' not found (the file may not exist)"
		);
		$this->assertFileVersionsCount($user, $fileId, $count);
	}

	/**
	 *
	 * @param string $path
	 * @param int $index
	 * @param string $user
	 * @param int $length
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the content length of file :path with version index :index for user :user in versions folder should be :length')]
	public function theContentLengthOfFileForUserInVersionsFolderIs(
		string $path,
		int $index,
		string $user,
		int $length
	): void {
		$user = $this->featureContext->getActualUsername($user);
		$fileId = $this->featureContext->getFileIdForPath($user, $path);
		Assert::assertNotNull(
			$fileId,
			__METHOD__
			. " fileid of file $path user $user not found (the file may not exist)"
		);
		$response = $this->listVersionFolder($user, $fileId, 1, ['d:getcontentlength']);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$response,
			__METHOD__
		);
		$xmlPart = $responseXmlObject->xpath("//d:prop/d:getcontentlength");
		Assert::assertEquals(
			$length,
			(int) $xmlPart[$index],
			"The content length of file $path with version $index for user $user was
			expected to be $length but the actual content length is $xmlPart[$index]"
		);
	}

	/**
	 * @param string $user
	 * @param string $path
	 * @param string $index
	 * @param string|NullCpuCoreFinder $spaceId
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function downloadVersion(
		string $user,
		string $path,
		string $index,
		?string $spaceId = null
	): ResponseInterface {
		$user = $this->featureContext->getActualUsername($user);
		$fileId = $this->featureContext->getFileIdForPath($user, $path, $spaceId);
		Assert::assertNotNull(
			$fileId,
			__METHOD__
			. " fileid of file $path user $user not found (the file may not exist)"
		);
		$index = (int)$index;
		$response = $this->listVersionFolder($user, $fileId, 1);
		if ($response->getStatusCode() === 403) {
			return $response;
		}
		$responseXmlObject = new SimpleXMLElement($response->getBody()->getContents());
		$xmlPart = $responseXmlObject->xpath("//d:response/d:href");
		if (!isset($xmlPart[$index])) {
			Assert::fail(
				'could not find version of path "' . $path . '" with index "' . $index . '"'
			);
		}
		// the href already contains the path
		$url = WebDavHelper::sanitizeUrl(
			$this->featureContext->getBaseUrlWithoutPath() . $xmlPart[$index]
		);
		return HttpRequestHelper::get(
			$url,
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user)
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $index
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :user downloads the version of file :path with the index :index')]
	public function userDownloadsVersion(string $user, string $path, string $index): void {
		$this->featureContext->setResponse($this->downloadVersion($user, $path, $index), $user);
	}

	/**
	 *
	 * @param string $index
	 * @param string $path
	 * @param string $user
	 * @param string $content
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the content of version index :index of file :path for user :user should be :content')]
	public function theContentOfVersionIndexOfFileForUserShouldBe(
		string $index,
		string $path,
		string $user,
		string $content
	): void {
		$response = $this->downloadVersion($user, $path, $index);
		$this->featureContext->theHTTPStatusCodeShouldBe("200", '', $response);
		$this->featureContext->checkDownloadedContentMatches($content, '', $response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $fileOrFileId
	 * @param string $path
	 *
	 * @return void
	 */
	#[When('/^user "([^"]*)" retrieves the meta information of (file|fileId) "([^"]*)" using the meta API$/')]
	public function userGetMetaInfo(string $user, string $fileOrFileId, string $path): void {
		$user = $this->featureContext->getActualUsername($user);
		$baseUrl = $this->featureContext->getBaseUrl();
		$password = $this->featureContext->getPasswordForUser($user);

		if ($fileOrFileId === "file") {
			$fileId = $this->featureContext->getFileIdForPath($user, $path);
			$metaPath = "/meta/$fileId/";
		} else {
			$metaPath = "/meta/$path/";
		}

		$body = '<?xml version="1.0" encoding="utf-8"?>
			<a:propfind xmlns:a="DAV:" xmlns:oc="http://owncloud.org/ns">
		    	<a:prop>
		    		<oc:meta-path-for-user />
		    	</a:prop>
			</a:propfind>';

		$response = WebDavHelper::makeDavRequest(
			$baseUrl,
			$user,
			$password,
			"PROPFIND",
			$metaPath,
			['Content-Type' => 'text/xml','Depth' => '0'],
			null,
			$this->featureContext->getStepLineRef(),
			$body,
			$this->featureContext->getDavPathVersion(),
			null
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * returns the result parsed into a SimpleXMLElement
	 * with a registered namespace with 'd' as prefix and 'DAV:' as namespace
	 *
	 * @param string $user
	 * @param string $fileId
	 * @param int $folderDepth
	 * @param string[]|null $properties
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function listVersionFolder(
		string $user,
		string $fileId,
		int $folderDepth,
		?array $properties = null
	): ResponseInterface {
		if (!$properties) {
			$properties = [
				'd:getetag'
			];
		}
		$user = $this->featureContext->getActualUsername($user);
		$password = $this->featureContext->getPasswordForUser($user);
		$response = WebDavHelper::propfind(
			$this->featureContext->getBaseUrl(),
			$user,
			$password,
			$fileId,
			$properties,
			$this->featureContext->getStepLineRef(),
			(string) $folderDepth,
			null,
			"versions"
		);
		return $response;
	}

	/**
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $space
	 *
	 * @return void
	 */
	#[When('user :user tries to get the number of versions of file :file from space :space')]
	public function userGetsTheNumberOfVersionsOfFileFromSpace(string $user, string $file, string $space): void {
		$fileId = $this->spacesContext->getFileId($user, $space, $file);
		$this->featureContext->setResponse(
			$this->featureContext->makeDavRequest(
				$user,
				"PROPFIND",
				$fileId,
				null,
				null,
				null,
				"versions"
			)
		);
	}
}
