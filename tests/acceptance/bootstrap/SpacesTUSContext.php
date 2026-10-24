<?php

declare(strict_types=1);

/**
 * @author Viktor Scharf <v.scharf@owncloud.com>
 * @copyright Copyright (c) 2022 Viktor Scharf v.scharf@owncloud.com
 */

use Behat\Behat\Context\Context;
use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use Behat\Gherkin\Node\TableNode;
use GuzzleHttp\Exception\GuzzleException;
use PHPUnit\Framework\Assert;
use TestHelpers\WebDavHelper;
use TestHelpers\BehatHelper;
use Behat\Step\Given;
use Behat\Step\Then;
use Behat\Step\When;

require_once 'bootstrap.php';

/**
 * Context for the TUS-specific steps using the Graph API
 */
class SpacesTUSContext implements Context {
	private FeatureContext $featureContext;
	private TUSContext $tusContext;
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
		$this->tusContext = BehatHelper::getContext($scope, $environment, 'TUSContext');
	}

	/**
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws Exception
	 * @throws GuzzleException
	 */
	#[Given('user :user has uploaded a file from :source to :destination via TUS inside of the space :spaceName using the WebDAV API')]
	public function userHasUploadedFileViaTusInSpace(
		string $user,
		string $source,
		string $destination,
		string $spaceName
	): void {
		$spaceId = $this->spacesContext->getSpaceIdByName($user, $spaceName);
		$this->tusContext->uploadFileUsingTus($user, $source, $destination, $spaceId);
		$this->featureContext->setLastUploadDeleteTime(\time());
	}

	/**
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	#[When('user :user uploads a file from :source to :destination via TUS inside of the space :spaceName using the WebDAV API')]
	public function userUploadsAFileViaTusInsideOfTheSpaceUsingTheWebdavApi(
		string $user,
		string $source,
		string $destination,
		string $spaceName
	): void {
		$spaceId = $this->spacesContext->getSpaceIdByName($user, $spaceName);
		$this->tusContext->uploadFileUsingTus($user, $source, $destination, $spaceId);
		$this->featureContext->setLastUploadDeleteTime(\time());
	}

	/**
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param TableNode $headers
	 *
	 * @return void
	 *
	 * @throws Exception
	 * @throws GuzzleException
	 */
	#[Given('user :user has created a new TUS resource in the space :spaceName with the following headers:')]
	public function userHasCreatedANewTusResourceForTheSpaceUsingTheWebdavApiWithTheseHeaders(
		string $user,
		string $spaceName,
		TableNode $headers
	): void {
		$spaceId = $this->spacesContext->getSpaceIdByName($user, $spaceName);
		$response = $this->tusContext->createNewTUSResourceWithHeaders($user, $headers, '', $spaceId);
		$this->featureContext->theHTTPStatusCodeShouldBe(201, "Expected response status code should be 201", $response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $content
	 * @param TableNode $headers
	 *
	 * @return void
	 *
	 * @throws Exception
	 * @throws GuzzleException
	 */
	#[When('user :user creates a new TUS resource for the space :spaceName with content :content using the WebDAV API with these headers:')]
	public function userCreatesANewTusResourceForTheSpaceUsingTheWebdavApiWithTheseHeaders(
		string $user,
		string $spaceName,
		string $content,
		TableNode $headers
	): void {
		$spaceId = $this->spacesContext->getSpaceIdByName($user, $spaceName);
		$response = $this->tusContext->createNewTUSResourceWithHeaders($user, $headers, $content, $spaceId);
		$this->featureContext->setResponse($response);
	}

	/**
	 * Uploads a file with content to the specified space using the TUS protocol via the WebDAV API.
	 *
	 * @param string $user
	 * @param string $content
	 * @param string $resource
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	private function uploadFileViaTus(string $user, string $content, string $resource, string $spaceName): void {
		$spaceId = $this->spacesContext->getSpaceIdByName($user, $spaceName);
		$tmpFile = $this->tusContext->writeDataToTempFile($content);
		try {
			$this->tusContext->uploadFileUsingTus(
				$user,
				\basename($tmpFile),
				$resource,
				$spaceId
			);
			$this->featureContext->setLastUploadDeleteTime(\time());
		} catch (Exception $e) {
			Assert::assertStringContainsString('Unable to create resource', (string)$e);
		}
		\unlink($tmpFile);
	}

	/**
	 *
	 * @param string $user
	 * @param string $content
	 * @param string $file
	 * @param string $destination
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	#[When('user :user uploads a file with content :content to :file inside federated share :destination via TUS using the WebDAV API')]
	public function userUploadsAFileWithContentToInsideFederatedShareViaTusUsingTheWebdavApi(
		string $user,
		string $content,
		string $file,
		string $destination
	): void {
		$remoteItemId = $this->spacesContext->getSharesRemoteItemId($user, $destination);
		$remoteItemId = \rawurlencode($remoteItemId);
		$tmpFile = $this->tusContext->writeDataToTempFile($content);
		$this->tusContext->uploadFileUsingTus(
			$user,
			\basename($tmpFile),
			$file,
			$remoteItemId
		);
		$this->featureContext->setLastUploadDeleteTime(\time());
		\unlink($tmpFile);
	}

	/**
	 *
	 * @param string $user
	 * @param string $content
	 * @param string $resource
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	#[When('user :user uploads a file with content :content to :resource via TUS inside of the space :spaceName using the WebDAV API')]
	public function userUploadsAFileWithContentToViaTusInsideOfTheSpaceUsingTheWebdavApi(
		string $user,
		string $content,
		string $resource,
		string $spaceName
	): void {
		$this->uploadFileViaTus($user, $content, $resource, $spaceName);
	}

	/**
	 *
	 * @param string $user
	 * @param string $content
	 * @param string $resource
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	#[Given('user :user has uploaded a file with content :content to :resource via TUS inside of the space :spaceName')]
	public function userHasUploadedAFileWithContentToViaTusInsideOfTheSpace(
		string $user,
		string $content,
		string $resource,
		string $spaceName
	): void {
		$this->uploadFileViaTus($user, $content, $resource, $spaceName);
	}

	/**
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $mtime Time in human-readable format is taken as input which is converted into milliseconds that is used by API
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws Exception
	 * @throws GuzzleException
	 */
	#[When('user :user uploads a file :source to :destination with mtime :mtime via TUS inside of the space :spaceName using the WebDAV API')]
	public function userUploadsAFileToWithMtimeViaTusInsideOfTheSpaceUsingTheWebdavApi(
		string $user,
		string $source,
		string $destination,
		string $mtime,
		string $spaceName
	): void {
		switch ($mtime) {
			case "today":
				$mtime = date('Y-m-d', strtotime('today'));
				break;
			case "yesterday":
				$mtime = date('Y-m-d', strtotime('yesterday'));
				break;
			case "lastWeek":
				$mtime = date('Y-m-d', strtotime('-7 days'));
				break;
			case "lastMonth":
				$mtime = date('Y-m-d', strtotime('first day of previous month'));
				break;
			case "lastYear":
				$mtime = date('Y-m' . '-01', strtotime('-1 year'));
				break;
			default:
		}
		$spaceId = $this->spacesContext->getSpaceIdByName($user, $spaceName);
		$mtime = new DateTime($mtime);
		$mtime = $mtime->format('U');
		$user = $this->featureContext->getActualUsername($user);
		$this->tusContext->uploadFileUsingTus(
			$user,
			$source,
			$destination,
			$spaceId,
			['mtime' => $mtime]
		);
		$this->featureContext->setLastUploadDeleteTime(\time());
	}

	/**
	 *
	 * @param string $user
	 * @param string $checksum
	 * @param string $offset
	 * @param string $content
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 * @codingStandardsIgnoreStart
	 */
	#[Given('user :user has uploaded file with checksum :checksum to the last created TUS Location with offset :offset and content :content via TUS inside of the space :spaceName using the WebDAV API')]
	public function userHasUploadedFileWithChecksumToTheLastCreatedTusLocationWithOffsetAndContentViaTusInsideOfTheSpaceUsingTheWebdavApi(
		// @codingStandardsIgnoreEnd
		string $user,
		string $checksum,
		string $offset,
		string $content,
		string $spaceName
	): void {
		$resourceLocation = $this->tusContext->getLastTusResourceLocation();
		$response = $this->tusContext->uploadChunkToTUSLocation($user, $resourceLocation, $offset, $content, $checksum);
		$this->featureContext->theHTTPStatusCodeShouldBe(204, "", $response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $checksum
	 * @param string $offset
	 * @param string $content
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 * @codingStandardsIgnoreStart
	 */
	#[When('user :user uploads file with checksum :checksum to the last created TUS Location with offset :offset and content :content via TUS inside of the space :spaceName using the WebDAV API')]
	public function userUploadsFileWithChecksumToTheLastCreatedTusLocationWithOffsetAndContentViaTusInsideOfTheSpaceUsingTheWebdavApi(
		// @codingStandardsIgnoreEnd
		string $user,
		string $checksum,
		string $offset,
		string $content,
		string $spaceName
	): void {
		$resourceLocation = $this->tusContext->getLastTusResourceLocation();
		$response = $this->tusContext->uploadChunkToTUSLocation($user, $resourceLocation, $offset, $content, $checksum);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $offset
	 * @param string $data
	 * @param string $checksum
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 * @codingStandardsIgnoreStart
	 */
	#[When('user :user sends a chunk to the last created TUS Location with offset :offset and data :data with checksum :checksum via TUS inside of the space :spaceName using the WebDAV API')]
	public function userSendsAChunkToTheLastCreatedTusLocationWithOffsetAndDataWithChecksumViaTusInsideOfTheSpaceUsingTheWebdavApi(
		// @codingStandardsIgnoreEnd
		string $user,
		string $offset,
		string $data,
		string $checksum,
		string $spaceName
	): void {
		$resourceLocation = $this->tusContext->getLastTusResourceLocation();
		$response = $this->tusContext->uploadChunkToTUSLocation($user, $resourceLocation, $offset, $data, $checksum);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $data
	 * @param TableNode $headers
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	#[When('user :user sends a chunk to the last created TUS Location with data :data with the following headers:')]
	public function userSendsAChunkToTheLastCreatedTusLocationWithDataInsideOfTheSpaceWithHeaders(
		string $user,
		string $data,
		TableNode $headers
	): void {
		$rows = $headers->getRowsHash();
		$resourceLocation = $this->tusContext->getLastTusResourceLocation();
		$response = $this->tusContext->uploadChunkToTUSLocation(
			$user,
			$resourceLocation,
			$rows['Upload-Offset'],
			$data,
			$rows['Upload-Checksum'],
			['Origin' => $rows['Origin']]
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $offset
	 * @param string $data
	 * @param string $checksum
	 * @param string $spaceName
	 * @param TableNode $headers
	 *
	 * @return void
	 * @throws GuzzleException
	 * @codingStandardsIgnoreStart
	 */
	#[When('user :user overwrites recently shared file with offset :offset and data :data with checksum :checksum via TUS inside of the space :spaceName using the WebDAV API with these headers:')]
	public function userOverwritesRecentlySharedFileWithOffsetAndDataWithChecksumViaTusInsideOfTheSpaceUsingTheWebdavApiWithTheseHeaders(
		// @codingStandardsIgnoreEnd
		string $user,
		string $offset,
		string $data,
		string $checksum,
		string $spaceName,
		TableNode $headers
	): void {
		$spaceId = $this->spacesContext->getSpaceIdByName($user, $spaceName);
		$createResponse = $this->tusContext->createNewTUSResource($user, $headers, $spaceId);
		$this->featureContext->theHTTPStatusCodeShouldBe(201, "", $createResponse);
		$resourceLocation = $this->tusContext->getLastTusResourceLocation();
		$response = $this->tusContext->uploadChunkToTUSLocation($user, $resourceLocation, $offset, $data, $checksum);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $resource
	 * @param string $spaceName
	 * @param string $mtime
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	#[Then('as :user the mtime of the file :resource in space :spaceName should be :mtime')]
	public function theMtimeOfTheFileInSpaceShouldBe(
		string $user,
		string $resource,
		string $spaceName,
		string $mtime
	): void {
		$spaceId = $this->spacesContext->getSpaceIdByName($user, $spaceName);
		$mtime = new DateTime($mtime);
		Assert::assertEquals(
			$mtime->format('U'),
			WebDavHelper::getMtimeOfResource(
				$this->featureContext->getActualUsername($user),
				$this->featureContext->getPasswordForUser($user),
				$this->featureContext->getBaseUrl(),
				$resource,
				$this->featureContext->getStepLineRef(),
				$this->featureContext->getDavPathVersion(),
				$spaceId,
			)
		);
	}
}
