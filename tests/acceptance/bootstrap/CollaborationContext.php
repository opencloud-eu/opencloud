<?php declare(strict_types=1);
/**
 * @author Amrita Shrestha <amrita@jankaritech.com>
 * @copyright Copyright (c) 2024 Amrita Shrestha <amrita@jankaritech.com>
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
use GuzzleHttp\Exception\GuzzleException;
use PHPUnit\Framework\Assert;
use TestHelpers\HttpRequestHelper;
use TestHelpers\WebDavHelper;
use TestHelpers\CollaborationHelper;
use TestHelpers\BehatHelper;
use Behat\Step\Given;
use Behat\Step\Then;
use Behat\Step\When;

/**
 * steps needed to re-configure OpenCloud server
 */
class CollaborationContext implements Context {
	private FeatureContext $featureContext;
	private SpacesContext $spacesContext;
	private string $lastAppOpenData;

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
	 * @param string $data
	 *
	 * @return void
	 */
	public function setLastAppOpenData(string $data): void {
		$this->lastAppOpenData = $data;
	}

	/**
	 * @return string
	 */
	public function getLastAppOpenData(): string {
		return $this->lastAppOpenData;
	}

	/**
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $space
	 * @param string $app
	 * @param string|null $viewMode
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	#[When('user :user checks the information of file :file of space :space using office :app')]
	#[When('user :user checks the information of file :file of space :space using office :app with view mode :view')]
	public function userChecksTheInformationOfFileOfSpaceUsingOffice(
		string $user,
		string $file,
		string $space,
		string $app,
		string $viewMode = null
	): void {
		$fileId = $this->spacesContext->getFileId($user, $space, $file);
		$response = \json_decode(
			CollaborationHelper::sendPOSTRequestToAppOpen(
				$fileId,
				$app,
				$this->featureContext->getActualUsername($user),
				$this->featureContext->getPasswordForUser($user),
				$this->featureContext->getBaseUrl(),
				$this->featureContext->getStepLineRef(),
				$viewMode
			)->getBody()->getContents()
		);

		$accessToken = $response->form_parameters->access_token;

		// Extract the WOPISrc from the app_url
		$parsedUrl = parse_url($response->app_url);
		parse_str($parsedUrl['query'], $queryParams);
		$wopiSrc = $queryParams['WOPISrc'];

		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$wopiSrc . "?access_token=$accessToken",
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $folder
	 * @param string $space
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('user :user creates a file :file inside folder :folder in space :space using wopi endpoint')]
	#[When('user :user tries to create a file :file inside folder :folder in space :space using wopi endpoint')]
	public function userCreatesFileInsideFolderInSpaceUsingWopiEndpoint(
		string $user,
		string $file,
		string $folder,
		string $space
	): void {
		$parentContainerId = $this->spacesContext->getResourceId($user, $space, $folder);
		$this->featureContext->setResponse(
			CollaborationHelper::createFile(
				$this->featureContext->getBaseUrl(),
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$parentContainerId,
				$file
			)
		);
	}

	/**
	 * @param string $file
	 * @param string $password
	 * @param string $folder
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function createFile(string $file, string $password, string $folder = ""): void {
		$token = $this->featureContext->shareNgGetLastCreatedLinkShareToken();
		$baseUrl = $this->featureContext->getBaseUrl();
		$davPath = WebDavHelper::getDavPath(WebDavHelper::DAV_VERSION_NEW, $token, "public-files");

		$responseXmlObject = HttpRequestHelper::getResponseXml(
			HttpRequestHelper::sendRequest(
				"$baseUrl/$davPath/$folder",
				$this->featureContext->getStepLineRef(),
				"PROPFIND",
				"public",
				$this->featureContext->getActualPassword($password)
			)
		);
		$xmlPart = $responseXmlObject->xpath("//d:prop/oc:fileid");
		$parentContainerId = (string) $xmlPart[0];

		$headers = [
			"Public-Token" => $token
		];
		$this->featureContext->setResponse(
			CollaborationHelper::createFile(
				$this->featureContext->getBaseUrl(),
				$this->featureContext->getStepLineRef(),
				"public",
				$this->featureContext->getActualPassword($password),
				$parentContainerId,
				$file,
				$headers
			)
		);
	}

	/**
	 *
	 * @param string $file
	 * @param string $password
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('the public creates a file :file inside the last shared public link folder with password :password using wopi endpoint')]
	#[When('the public tries to create a file :file inside the last shared public link folder with password :password using wopi endpoint')]
	public function thePublicCreatesAFileInsideTheLastSharedPublicLinkFolderWithPasswordUsingWopiEndpoint(
		string $file,
		string $password
	): void {
		$this->createFile($file, $password);
	}

	/**
	 *
	 * @param string $file
	 * @param string $folder
	 * @param string $password
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('the public creates a file :file inside folder :folder in the last shared public link space with password :password using wopi endpoint')]
	#[When('the public tries to create a file :file inside folder :folder in the last shared public link space with password :password using wopi endpoint')]
	public function thePublicCreatesAFileInsideFolderInTheLastSharedPublicLinkSpaceWithPasswordUsingWopiEndpoint(
		string $file,
		string $folder,
		string $password
	): void {
		$this->createFile($file, $password, $folder);
	}

	/**
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $space
	 * @param string $app
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	#[When('user :user tries to check the information of file :file of space :space using office :app with invalid file-id')]
	public function userTriesToCheckTheInformationOfFileOfSpaceUsingOfficeWithInvalidFileId(
		string $user,
		string $file,
		string $space,
		string $app
	): void {
		$response = \json_decode(
			CollaborationHelper::sendPOSTRequestToAppOpen(
				$this->spacesContext->getFileId($user, $space, $file),
				$app,
				$this->featureContext->getActualUsername($user),
				$this->featureContext->getPasswordForUser($user),
				$this->featureContext->getBaseUrl(),
				$this->featureContext->getStepLineRef()
			)->getBody()->getContents()
		);
		$accessToken = $response->form_parameters->access_token;

		// Extract the WOPISrc from the app_url
		$parsedUrl = parse_url($response->app_url);
		parse_str($parsedUrl['query'], $queryParams);
		$wopiSrc = $queryParams['WOPISrc'];
		$position = strpos($wopiSrc, '/files/') + \strlen('/files/');

		// Extract the base URL up to and including '/files/'
		$fullUrl = substr($wopiSrc, 0, $position) . WebDavHelper::generateUUIDv4();
		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$fullUrl . "?access_token=$accessToken",
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $file
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('user :user tries to create a file :file inside deleted folder using wopi endpoint')]
	public function userTriesToCreateAFileInsideDeletedFolderUsingWopiEndpoint(string $user, string $file): void {
		$parentContainerId = $this->featureContext->getStoredFileID();
		$this->featureContext->setResponse(
			CollaborationHelper::createFile(
				$this->featureContext->getBaseUrl(),
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$parentContainerId,
				$file
			)
		);
	}

	/**
	 *
	 * @param string $user
	 * @param TableNode $properties
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Given('user :user has sent the following app-open request:')]
	public function userHasSentTheFollowingAppOpenRequest(string $user, TableNode $properties): void {
		$rows = $properties->getRowsHash();
		$appResponse = CollaborationHelper::sendPOSTRequestToAppOpen(
			$this->spacesContext->getFileId($user, $rows['space'], $rows['resource']),
			$rows['app'],
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef()
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, '', $appResponse);
		$this->setLastAppOpenData($appResponse->getBody()->getContents());
	}

	/**
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('user :user tries to get the information of the last opened file using wopi endpoint')]
	#[When('user :user gets the information of the last opened file using wopi endpoint')]
	public function userGetsTheInformationOfTheLastOpenedFileUsingWopiEndpoint(string $user): void {
		$response = json_decode($this->getLastAppOpenData());
		$accessToken = $response->form_parameters->access_token;

		// Extract the WOPISrc from the app_url
		$parsedUrl = parse_url($response->app_url);
		parse_str($parsedUrl['query'], $queryParams);
		$wopiSrc = $queryParams['WOPISrc'];

		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$wopiSrc . "?access_token=$accessToken",
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 *
	 * @param string    $shouldOrNot
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('/^the response (should|should not) contain the following MIME types:$/')]
	public function theFollowingMimeTypesShouldExistForUser(string $shouldOrNot, TableNode $table): void {
		$rows = $table->getRows();
		$responseArray = $this->featureContext->getJsonDecodedResponse(
			$this->featureContext->getResponse()
		)['mime-types'];
		$mimeTypes = \array_column($responseArray, 'mime_type');
		foreach ($rows as $row) {
			if ($shouldOrNot === "should not") {
				Assert::assertFalse(
					\in_array($row[0], $mimeTypes),
					"the response should not contain the mimetype $row[0].\nMime Types found in response:\n"
					. print_r($mimeTypes, true)
				);
			} else {
				Assert::assertTrue(
					\in_array($row[0], $mimeTypes),
					"the response does not contain the mimetype $row[0].\nMime Types found in response:\n"
					. print_r($mimeTypes, true)
				);
			}
		}
	}

	/**
	 *
	 * @param string $app
	 * @param TableNode $table
	 *
	 * @return void
	 */
	#[Then('the app list response should contain the following template information for office :app:')]
	public function theAppListResponseShouldContainTheFollowingTemplateInformationForOffice(
		string $app,
		TableNode $table
	): void {
		$responseArray = $this->featureContext->getJsonDecodedResponse($this->featureContext->getResponse());

		Assert::assertArrayHasKey(
			"mime-types",
			$responseArray,
			"Expected 'mime-types' in the response but not found.\n" . print_r($responseArray, true)
		);

		$mimeTypes = $responseArray['mime-types'];

		$mimeTypeMap = [];
		foreach ($mimeTypes as $mimeType) {
			$mimeTypeMap[$mimeType['mime_type']] = $mimeType;
		}

		foreach ($table->getColumnsHash() as $row) {
			Assert::assertArrayHasKey(
				$row['mime-type'],
				$mimeTypeMap,
				"Expected mime-type '{$row['mime-type']}' to exist in the response but it doesn't.\n"
				. print_r($mimeTypeMap, true)
			);

			$mimeType = $mimeTypeMap[$row['mime-type']];
			$found = false;

			foreach ($mimeType['app_providers'] as $provider) {
				if ($provider['name'] === $app && isset($row['target-extension'])) {
					Assert::assertSame(
						$row['target-extension'],
						$provider['target_ext'],
						"Expected 'target_ext' for $app to be '{$row['target-extension']}'"
						. " but found '{$provider['target_ext']}'"
					);
					$found = true;
					break;
				}
			}

			if (!$found) {
				Assert::fail(
					"Expected response to contain app-provider '$app' with target-extension "
					. "'{$row['target-extension']}' for mime-type '{$row['mime-type']}',"
					. " but no matching provider was found.\n App Providers Found: "
					. print_r($mimeType['app_providers'], true)
				);
			}
		}
	}

	/**
	 *
	 * @param string $user
	 * @param string $file
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	#[When('user :user has created a file :file using wopi endpoint')]
	public function userHasCreatedAFileInSpaceUsingWopiEndpoint(string $user, string $file): string {
		$parentContainerId = $this->featureContext->getFileIdForPath($user, "/");
		$response = CollaborationHelper::createFile(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$parentContainerId,
			$file
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
		$decodedResponse = $this->featureContext->getJsonDecodedResponseBodyContent($response);
		return $decodedResponse->file_id;
	}
}
