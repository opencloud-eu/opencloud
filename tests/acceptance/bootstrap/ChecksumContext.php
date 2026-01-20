<?php declare(strict_types=1);
/**
 * @author Roeland Jago Douma <rullzer@owncloud.com>
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
use PHPUnit\Framework\Assert;
use Psr\Http\Message\ResponseInterface;
use TestHelpers\WebDavHelper;
use TestHelpers\BehatHelper;
use TestHelpers\UploadHelper;

require_once 'bootstrap.php';

/**
 * Checksum functions
 */
class ChecksumContext implements Context {
	private FeatureContext $featureContext;

	/**
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $checksum
	 *
	 * @return ResponseInterface
	 */
	public function uploadFileToWithChecksumUsingTheAPI(
		string $user,
		string  $source,
		string  $destination,
		string  $checksum
	): ResponseInterface {
		$file = \file_get_contents(
			UploadHelper::getAcceptanceTestsDir() . $source
		);
		return $this->featureContext->makeDavRequest(
			$user,
			'PUT',
			$destination,
			['OC-Checksum' => $checksum],
			$file
		);
	}

	/**
	 * @When user :user uploads file :source to :destination with checksum :checksum using the WebDAV API
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $checksum
	 *
	 * @return void
	 */
	public function userUploadsFileToWithChecksumUsingTheAPI(
		string $user,
		string  $source,
		string  $destination,
		string  $checksum
	): void {
		$this->featureContext->setResponse(
			$this->uploadFileToWithChecksumUsingTheAPI($user, $source, $destination, $checksum)
		);
	}

	/**
	 * @Given user :user has uploaded file :source to :destination with checksum :checksum
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $checksum
	 *
	 * @return void
	 */
	public function userHasUploadedFileToWithChecksumUsingTheAPI(
		string $user,
		string $source,
		string $destination,
		string $checksum
	): void {
		$user = $this->featureContext->getActualUsername($user);
		$response = $this->uploadFileToWithChecksumUsingTheAPI(
			$user,
			$source,
			$destination,
			$checksum
		);
		$this->featureContext->theHTTPStatusCodeShouldBe([201,204], '', $response);
	}

	/**
	 * @param string $user
	 * @param string $content
	 * @param string $checksum
	 * @param string $destination
	 *
	 * @return ResponseInterface
	 */
	public function uploadFileWithContentAndChecksumToUsingTheAPI(
		string $user,
		string $content,
		string $checksum,
		string $destination
	): ResponseInterface {
		return $this->featureContext->makeDavRequest(
			$user,
			'PUT',
			$destination,
			['OC-Checksum' => $checksum],
			$content
		);
	}

	/**
	 * @Given user :user has uploaded file with content :content and checksum :checksum to :destination
	 *
	 * @param string $user
	 * @param string $content
	 * @param string $checksum
	 * @param string $destination
	 *
	 * @return void
	 */
	public function userHasUploadedFileWithContentAndChecksumToUsingTheAPI(
		string $user,
		string $content,
		string $checksum,
		string $destination
	): void {
		$user = $this->featureContext->getActualUsername($user);
		$response = $this->uploadFileWithContentAndChecksumToUsingTheAPI($user, $content, $checksum, $destination);
		$this->featureContext->theHTTPStatusCodeShouldBe(201, '', $response);
	}

	/**
	 * @When user :user requests the checksum of :path via propfind
	 *
	 * @param string $user
	 * @param string $path
	 *
	 * @return void
	 */
	public function userRequestsTheChecksumOfViaPropfind(string $user, string $path): void {
		$this->featureContext->setResponse($this->propfindResourceChecksum($user, $path));
	}

	/**
	 * @param string $user
	 * @param string $path
	 * @param string|null $spaceId
	 *
	 * @return ResponseInterface
	 */
	public function propfindResourceChecksum(string $user, string $path, ?string $spaceId = null): ResponseInterface {
		$user = $this->featureContext->getActualUsername($user);
		$body = '<?xml version="1.0"?>
			<d:propfind  xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
			  <d:prop>
				<oc:checksums />
			  </d:prop>
			</d:propfind>';
		$password = $this->featureContext->getPasswordForUser($user);
		return WebDavHelper::makeDavRequest(
			$this->featureContext->getBaseUrl(),
			$user,
			$password,
			'PROPFIND',
			$path,
			null,
			$spaceId,
			$this->featureContext->getStepLineRef(),
			$body,
			$this->featureContext->getDavPathVersion()
		);
	}

	/**
	 * @Then the webdav checksum should match :expectedChecksum
	 *
	 * @param string $expectedChecksum
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theWebdavChecksumShouldMatch(string $expectedChecksum): void {
		$bodyContents = $this->featureContext->getResponse()->getBody()->getContents();
		$this->validateChecksum($bodyContents, $expectedChecksum);
	}

	/**
	 * @Then as user :user the webdav checksum of :path via propfind should match :expectedChecksum
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $expectedChecksum
	 *
	 * @return void
	 * @throws Exception
	 */
	public function asUserTheWebdavChecksumOfPathViaPropfindShouldMatch(
		string $user,
		string $path,
		string $expectedChecksum
	): void {
		$user = $this->featureContext->getActualUsername($user);
		$resource = $this->propfindResourceChecksum($user, $path);
		$bodyContents = $resource->getBody()->getContents();
		$this->validateChecksum($bodyContents, $expectedChecksum);
	}
	/**
	 * @param string $bodyContents
	 * @param string $expectedChecksum
	 *
	 * @return void
	 */
	public function validateChecksum(string $bodyContents, string $expectedChecksum): void {
		$service = new Sabre\Xml\Service();
		$parsed = $service->parse($bodyContents);

		/*
		 * Fetch the checksum array
		 * The checksums are way down in the array:
		 * $checksums = $parsed[0]['value'][1]['value'][0]['value'][0];
		 * And inside is the actual checksum string:
		 * $checksums['value'][0]['value']
		 * The Asserts below check the existence of the expected key at every level
		 * of the nested array. This helps to see what happened if a test fails
		 * because the response structure is not as expected.
		 */

		Assert::assertIsArray(
			$parsed,
			__METHOD__
			. " could not parse response as XML. Expected parsed XML to be an array but found " . $bodyContents
		);
		Assert::assertArrayHasKey(
			0,
			$parsed,
			__METHOD__ . " parsed XML does not have key 0"
		);
		$parsed0 = $parsed[0];
		Assert::assertArrayHasKey(
			'value',
			$parsed0,
			__METHOD__ . " parsed XML parsed0 does not have key value"
		);
		$parsed0Value = $parsed0['value'];
		Assert::assertArrayHasKey(
			1,
			$parsed0Value,
			__METHOD__ . " parsed XML parsed0Value does not have key 1"
		);
		$parsed0Value1 = $parsed0Value[1];
		Assert::assertArrayHasKey(
			'value',
			$parsed0Value1,
			__METHOD__ . " parsed XML parsed0Value1 does not have key value after key 1"
		);
		$parsed0Value1Value = $parsed0Value1['value'];
		Assert::assertArrayHasKey(
			0,
			$parsed0Value1Value,
			__METHOD__ . " parsed XML parsed0Value1Value does not have key 0"
		);
		$parsed0Value1Value0 = $parsed0Value1Value[0];
		Assert::assertArrayHasKey(
			'value',
			$parsed0Value1Value0,
			__METHOD__ . " parsed XML parsed0Value1Value0 does not have key value"
		);
		$parsed0Value1Value0Value = $parsed0Value1Value0['value'];
		Assert::assertArrayHasKey(
			0,
			$parsed0Value1Value0Value,
			__METHOD__ . " parsed XML parsed0Value1Value0Value does not have key 0"
		);
		$checksums = $parsed0Value1Value0Value[0];
		Assert::assertArrayHasKey(
			'value',
			$checksums,
			__METHOD__ . " parsed XML checksums does not have key value"
		);
		$checksumsValue = $checksums['value'];
		Assert::assertArrayHasKey(
			0,
			$checksumsValue,
			__METHOD__ . " parsed XML checksumsValue does not have key 0"
		);
		$checksumsValue0 = $checksumsValue[0];
		Assert::assertArrayHasKey(
			'value',
			$checksumsValue0,
			__METHOD__ . " parsed XML checksumsValue0 does not have key value"
		);
		$actualChecksum = $checksumsValue0['value'];
		Assert::assertEquals(
			$expectedChecksum,
			$actualChecksum,
			"Expected: webDav checksum should be $expectedChecksum but got $actualChecksum"
		);
	}

	/**
	 * @Then the header checksum should match :expectedChecksum
	 *
	 * @param string $expectedChecksum
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theHeaderChecksumShouldMatch(string $expectedChecksum): void {
		$headerChecksums
			= $this->featureContext->getResponse()->getHeader('OC-Checksum');

		Assert::assertIsArray(
			$headerChecksums,
			__METHOD__ . " getHeader('OC-Checksum') did not return an array"
		);

		Assert::assertNotEmpty(
			$headerChecksums,
			__METHOD__ . " getHeader('OC-Checksum') returned an empty array. No checksum header was found."
		);

		$checksumCount = \count($headerChecksums);

		Assert::assertTrue(
			$checksumCount === 1,
			__METHOD__ . " Expected 1 checksum in the header but found $checksumCount checksums"
		);

		$headerChecksum
			= $headerChecksums[0];
		Assert::assertEquals(
			$expectedChecksum,
			$headerChecksum,
			"Expected: header checksum should match $expectedChecksum but got $headerChecksum"
		);
	}

	/**
	 * @Then the header checksum when user :arg1 downloads file :arg2 using the WebDAV API should match :arg3
	 *
	 * @param string $user
	 * @param string $fileName
	 * @param string $expectedChecksum
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theHeaderChecksumWhenUserDownloadsFileUsingTheWebdavApiShouldMatch(
		string $user,
		string $fileName,
		string $expectedChecksum
	): void {
		$response = $this->featureContext->downloadFileAsUserUsingPassword($user, $fileName);
		$headerChecksums = $response->getHeader('OC-Checksum');

		Assert::assertIsArray(
			$headerChecksums,
			__METHOD__ . " getHeader('OC-Checksum') did not return an array"
		);

		Assert::assertNotEmpty(
			$headerChecksums,
			__METHOD__ . " getHeader('OC-Checksum') returned an empty array. No checksum header was found."
		);

		$checksumCount = \count($headerChecksums);

		Assert::assertTrue(
			$checksumCount === 1,
			__METHOD__ . " Expected 1 checksum in the header but found $checksumCount checksums"
		);

		$headerChecksum
			= $headerChecksums[0];
		Assert::assertEquals(
			$expectedChecksum,
			$headerChecksum,
			"Expected: header checksum should match $expectedChecksum but got $headerChecksum"
		);
	}

	/**
	 * @param string $user
	 * @param int $num
	 * @param int $total
	 * @param string $data
	 * @param string $destination
	 * @param string $expectedChecksum
	 *
	 * @return ResponseInterface
	 */
	public function uploadChunkFileOfWithToWithChecksum(
		string $user,
		int $num,
		int $total,
		string $data,
		string $destination,
		string $expectedChecksum
	): ResponseInterface {
		$user = $this->featureContext->getActualUsername($user);
		$num -= 1;
		$file = "$destination-chunking-42-$total-$num";
		return $this->featureContext->makeDavRequest(
			$user,
			'PUT',
			$file,
			['OC-Checksum' => $expectedChecksum, 'OC-Chunked' => '1'],
			$data
		);
	}

	/**
	 * @When user :user uploads chunk file :num of :total with :data to :destination with checksum :expectedChecksum using the WebDAV API
	 *
	 * @param string $user
	 * @param int $num
	 * @param int $total
	 * @param string $data
	 * @param string $destination
	 * @param string $expectedChecksum
	 *
	 * @return void
	 */
	public function userUploadsChunkFileOfWithToWithChecksum(
		string $user,
		int $num,
		int $total,
		string $data,
		string $destination,
		string $expectedChecksum
	): void {
		$response = $this->uploadChunkFileOfWithToWithChecksum(
			$user,
			$num,
			$total,
			$data,
			$destination,
			$expectedChecksum
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @Given user :user has uploaded chunk file :num of :total with :data to :destination with checksum :expectedChecksum
	 *
	 * @param string $user
	 * @param int $num
	 * @param int $total
	 * @param string $data
	 * @param string $destination
	 * @param string $expectedChecksum
	 *
	 * @return void
	 */
	public function userHasUploadedChunkFileOfWithToWithChecksum(
		string $user,
		int $num,
		int $total,
		string $data,
		string $destination,
		string $expectedChecksum
	): void {
		$response = $this->uploadChunkFileOfWithToWithChecksum(
			$user,
			$num,
			$total,
			$data,
			$destination,
			$expectedChecksum
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			[201, 206],
			'',
			$response
		);
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
}
