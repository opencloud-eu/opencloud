<?php declare(strict_types=1);
/**
 * @author Joas Schilling <coding@schilljs.com>
 * @author Sergio Bertolin <sbertolin@owncloud.com>
 * @author Phillip Davis <phil@jankaritech.com>
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
use Behat\Gherkin\Node\PyStringNode;
use Psr\Http\Message\ResponseInterface;
use PHPUnit\Framework\Assert;
use TestHelpers\OcsApiHelper;
use TestHelpers\BehatHelper;
use TestHelpers\HttpRequestHelper;

require_once 'bootstrap.php';

/**
 * Capabilities context.
 */
class CapabilitiesContext implements Context {
	private FeatureContext $featureContext;

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
	 * @param string $username
	 * @param boolean $formatJson // if true then formats the response in json
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userGetsCapabilities(string $username, ?bool $formatJson = false): ResponseInterface {
		$user = $this->featureContext->getActualUsername($username);
		$password = $this->featureContext->getPasswordForUser($user);
		return OcsApiHelper::sendRequest(
			$this->featureContext->getBaseUrl(),
			$user,
			$password,
			'GET',
			'/cloud/capabilities' . ($formatJson ? '?format=json' : ''),
			$this->featureContext->getStepLineRef(),
			[],
			$this->featureContext->getOcsApiVersion()
		);
	}

	/**
	 * @return string
	 * @throws Exception|GuzzleException
	 */
	public function getAdminUsernameForCapabilitiesCheck(): string {
		if (\TestHelpers\OcHelper::isTestingOnReva()) {
			// When testing on reva we don't have a user called "admin" to use
			// to access the capabilities. So create an ordinary user on-the-fly
			// with a default password. That user should be able to get a
			// capabilities response that the test can process.
			$adminUsername = "PseudoAdminForRevaTest";
			$createdUsers = $this->featureContext->getCreatedUsers();
			if (!\array_key_exists($adminUsername, $createdUsers)) {
				$this->featureContext->userHasBeenCreated(["userName" => $adminUsername]);
			}
		} else {
			$adminUsername = $this->featureContext->getAdminUsername();
		}
		return $adminUsername;
	}

	/**
	 * @param SimpleXMLElement $xml of the capabilities
	 * @param string $capabilitiesApp the "app" name in the capabilities response
	 * @param string $capabilitiesPath the path to the element
	 *
	 * @return string
	 */
	public function getParameterValueFromXml(
		SimpleXMLElement $xml,
		string $capabilitiesApp,
		string $capabilitiesPath
	): string {
		$path_to_element = \explode('@@@', $capabilitiesPath);
		$answeredValue = $xml->{$capabilitiesApp};
		foreach ($path_to_element as $element) {
			$nameIndexParts = \explode('[', $element);
			if (isset($nameIndexParts[1])) {
				// This part of the path should be something like "some_element[1]"
				// Separately extract the name and the index
				$name = $nameIndexParts[0];
				$index = (int) \explode(']', $nameIndexParts[1])[0];
				// and use those to construct the reference into the next XML level
				$answeredValue = $answeredValue->{$name}[$index];
			} else {
				if ($element !== "") {
					$answeredValue = $answeredValue->{$element};
				}
			}
		}

		return (string) $answeredValue;
	}

	/**
	 * @When user :username retrieves the capabilities using the capabilities API
	 *
	 * @param string $username
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userRetrievesCapabilities(string $username): void {
		$user = $this->featureContext->getActualUsername($username);
		$this->featureContext->setResponse($this->userGetsCapabilities($user, true));
	}

	/**
	 * @When the administrator retrieves the capabilities using the capabilities API
	 *
	 * @return void
	 */
	public function theAdministratorGetsCapabilities(): void {
		$user = $this->getAdminUsernameForCapabilitiesCheck();
		$this->featureContext->setResponse($this->userGetsCapabilities($user, true));
	}

	/**
	 * @Then the major-minor-micro version data in the response should match the version string
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkVersionMajorMinorMicroResponse(): void {
		$jsonResponse = $this->featureContext->getJsonDecodedResponseBodyContent();
		$versionData = $jsonResponse->ocs->data->version;
		$versionString = (string) $versionData->string;
		// We expect that versionString will be in a format like "10.9.2 beta" or "10.9.2-alpha" or "10.9.2"
		$result = \preg_match('/^[0-9]+\.[0-9]+\.[0-9]+/', $versionString, $matches);
		Assert::assertSame(
			1,
			$result,
			__METHOD__ . " version string '$versionString' does not start with a semver version"
		);
		// semVerParts should have an array with the 3 semver components of the version, e.g. "1", "9" and "2".
		$semVerParts = \explode('.', $matches[0]);
		$expectedMajor = $semVerParts[0];
		$expectedMinor = $semVerParts[1];
		$expectedMicro = $semVerParts[2];
		$actualMajor = (string) $versionData->major;
		$actualMinor = (string) $versionData->minor;
		$actualMicro = (string) $versionData->micro;
		Assert::assertSame(
			$expectedMajor,
			$actualMajor,
			__METHOD__ . "'major' data item does not match with major version in string '$versionString'"
		);
		Assert::assertSame(
			$expectedMinor,
			$actualMinor,
			__METHOD__ . "'minor' data item does not match with minor version in string '$versionString'"
		);
		Assert::assertSame(
			$expectedMicro,
			$actualMicro,
			__METHOD__ . "'micro' data item does not match with micro (patch) version in string '$versionString'"
		);
	}

	/**
	 * @Then the status.php response should include
	 *
	 * @param PyStringNode $jsonExpected
	 *
	 * @return void
	 * @throws Exception
	 */
	public function statusPhpRespondedShouldMatch(PyStringNode $jsonExpected): void {
		$jsonExpectedDecoded = \json_decode($jsonExpected->getRaw(), true);
		$jsonRespondedDecoded = $this->featureContext->getJsonDecodedResponse();

		$response = $this->userGetsCapabilities($this->getAdminUsernameForCapabilitiesCheck());
		$this->featureContext->theHTTPStatusCodeShouldBe(200, '', $response);

		$responseXmlObject = HttpRequestHelper::getResponseXml($response, __METHOD__)->data->capabilities;
		$edition = $this->getParameterValueFromXml(
			$responseXmlObject,
			'core',
			'status@@@edition'
		);

		$product = $this->getParameterValueFromXml(
			$responseXmlObject,
			'core',
			'status@@@product'
		);
		if (!\strlen($product)) {
			Assert::fail(
				"Cannot get product from core capabilities"
			);
		}

		$productName = $this->getParameterValueFromXml(
			$responseXmlObject,
			'core',
			'status@@@productname'
		);

		if (!\strlen($productName)) {
			Assert::fail(
				"Cannot get productname from core capabilities"
			);
		}

		$jsonExpectedDecoded['edition'] = $edition;
		$jsonExpectedDecoded['product'] = $product;
		$jsonExpectedDecoded['productname'] = $productName;

		// We are on OpenCloud or reva or some other implementation. We cannot do "occ status".
		// So get the expected version values by looking in the capabilities response.
		$version = $this->getParameterValueFromXml(
			$responseXmlObject,
			'core',
			'status@@@version'
		);

		if (!\strlen($version)) {
			Assert::fail(
				"Cannot get version from core capabilities"
			);
		}

		$versionString = $this->getParameterValueFromXml(
			$responseXmlObject,
			'core',
			'status@@@versionstring'
		);

		if (!\strlen($versionString)) {
			Assert::fail(
				"Cannot get versionstring from core capabilities"
			);
		}

		$jsonExpectedDecoded['version'] = $version;
		$jsonExpectedDecoded['versionstring'] = $versionString;
		$errorMessage = "";
		$errorFound = false;
		foreach ($jsonExpectedDecoded as $key => $expectedValue) {
			if (\array_key_exists($key, $jsonRespondedDecoded)) {
				$actualValue = $jsonRespondedDecoded[$key];
				if ($actualValue !== $expectedValue) {
					$errorMessage .= "$key expected value was $expectedValue but actual value was $actualValue\n";
					$errorFound = true;
				}
			} else {
				$errorMessage .= "$key was not found in the status response\n";
				$errorFound = true;
			}
		}
		Assert::assertFalse($errorFound, $errorMessage);
		// We have checked that the status.php response has data that matches up with
		// data found in the capabilities response and/or the "occ status" command output.
		// But the output might be reported wrongly in all of these in the same way.
		// So check that the values also seem "reasonable".
		$version = $jsonExpectedDecoded['version'];
		$versionString = $jsonExpectedDecoded['versionstring'];
		Assert::assertMatchesRegularExpression(
			"/^\d+\.\d+\.\d+\.\d+$/",
			$version,
			"version should be in a form like 10.9.8.1 but is $version"
		);
		if (\preg_match("/^(\d+\.\d+\.\d+)\.\d+(-[0-9A-Za-z-]+)?(\+[0-9A-Za-z-]+)?$/", $version, $matches)) {
			// We should have matched something like 10.9.8 - the first 3 numbers in the version.
			// Ignore pre-releases and meta information
			Assert::assertArrayHasKey(
				1,
				$matches,
				"version $version could not match the pattern Major.Minor.Patch"
			);
			$majorMinorPatchVersion = $matches[1];
		} else {
			Assert::fail("version '$version' does not start in a form like 10.9.8");
		}
		Assert::assertStringStartsWith(
			$majorMinorPatchVersion,
			$versionString,
			"version string should start with $majorMinorPatchVersion but is $versionString"
		);
	}
}
