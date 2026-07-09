<?php declare(strict_types=1);
/**
 * @author Artur Neumann <artur@jankaritech.com>
 * @copyright Copyright (c) 2019, ownCloud GmbH
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
use TestHelpers\Asserts\WebDav as WebDavTest;
use TestHelpers\WebDavHelper;
use TestHelpers\BehatHelper;
use TestHelpers\HttpRequestHelper;
use Behat\Step\Given;
use Behat\Step\Then;
use Behat\Step\When;

require_once 'bootstrap.php';

/**
 * Steps that relate to managing file/folder properties via WebDav
 */
class WebDavPropertiesContext implements Context {
	private FeatureContext $featureContext;

	/**
	 * @var array map with user as key and another map as value,
	 *            which has path as key and etag as value
	 */
	private array $storedETAG = [];

	/**
	 *
	 * @param string $user
	 * @param string $path
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('/^user "([^"]*)" gets the properties of (?:file|folder|entry) "([^"]*)" using the WebDAV API$/')]
	public function userGetsThePropertiesOfFolder(
		string $user,
		string $path
	): void {
		$response = $this->featureContext->listFolder(
			$user,
			$path,
			'0'
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $depth
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('/^user "([^"]*)" gets the properties of (?:file|folder|entry) "([^"]*)" with depth (\\d+) using the WebDAV API$/')]
	public function userGetsThePropertiesOfFolderWithDepth(
		string $user,
		string $path,
		string $depth
	): void {
		$response = $this->featureContext->listFolder(
			$user,
			$path,
			$depth
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @param string $user
	 * @param string $path
	 * @param string|null $spaceId
	 * @param TableNode|null $propertiesTable
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function getPropertiesOfFolder(
		string $user,
		string $path,
		?string $spaceId,
		TableNode $propertiesTable
	): ResponseInterface {
		$user = $this->featureContext->getActualUsername($user);
		$properties = null;
		$this->featureContext->verifyTableNodeColumns($propertiesTable, ["propertyName"]);
		$this->featureContext->verifyTableNodeColumnsCount($propertiesTable, 1);
		foreach ($propertiesTable->getColumnsHash() as $row) {
			$properties[] = $row["propertyName"];
		}
		return $this->featureContext->listFolder(
			$user,
			$path,
			"1",
			$properties,
			$spaceId,
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $path
	 * @param TableNode|null $propertiesTable
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('/^user "([^"]*)" gets the following properties of (?:file|folder|entry) "([^"]*)" using the WebDAV API$/')]
	public function userGetsFollowingPropertiesOfEntryUsingWebDavApi(
		string $user,
		string $path,
		TableNode $propertiesTable
	): void {
		$response = $this->getPropertiesOfFolder($user, $path, null, $propertiesTable);
		$this->featureContext->setResponse($response);
		$this->featureContext->pushToLastStatusCodesArrays();
	}

	/**
	 *
	 * @param string $username
	 * @param string $path
	 * @param TableNode|null $propertiesTable with following columns with column header as:
	 *                                        property: name of prop to be set
	 *                                        value: value of prop to be set
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Given('/^user "([^"]*)" has set the following properties to (?:file|folder|entry) "([^"]*)" using the WebDav API$/')]
	public function userHasSetFollowingPropertiesUsingProppatch(
		string $username,
		string $path,
		TableNode $propertiesTable
	) {
		$username = $this->featureContext->getActualUsername($username);
		$this->featureContext->verifyTableNodeColumns($propertiesTable, ['propertyName', 'propertyValue']);
		$properties = $propertiesTable->getColumnsHash();
		$response = WebDavHelper::proppatchWithMultipleProps(
			$this->featureContext->getBaseUrl(),
			$username,
			$this->featureContext->getPasswordForUser($username),
			$path,
			$properties,
			$this->featureContext->getStepLineRef(),
			$this->featureContext->getDavPathVersion()
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(207, "", $response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $propertyName
	 * @param string $path
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :user gets a custom property :propertyName of file :path')]
	public function userGetsCustomPropertyOfFile(
		string $user,
		string $propertyName,
		string $path
	): void {
		$user = $this->featureContext->getActualUsername($user);
		$properties = [$propertyName];
		$response = WebDavHelper::propfind(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getUserPassword($user),
			$path,
			$properties,
			$this->featureContext->getStepLineRef(),
			"0",
			null,
			"files",
			$this->featureContext->getDavPathVersion()
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $propertyName
	 * @param string $namespace namespace in the form of "x1='http://whatever.org/ns'"
	 * @param string $path
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :user gets a custom property :propertyName with namespace :namespace of file :path')]
	public function userGetsPropertiesWithNamespaceOfFile(
		string $user,
		string $propertyName,
		string $namespace,
		string $path
	): void {
		$user = $this->featureContext->getActualUsername($user);
		$properties = [
			$namespace => $propertyName
		];
		$response = WebDavHelper::propfind(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getUserPassword($user),
			$path,
			$properties,
			$this->featureContext->getStepLineRef(),
			"0",
			null,
			"files",
			$this->featureContext->getDavPathVersion()
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @param string $path
	 * @param TableNode $propertiesTable
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function getPropertiesOfEntryFromLastLinkShare(string $path, TableNode $propertiesTable): ResponseInterface {
		$token = ($this->featureContext->isUsingSharingNG())
		? $this->featureContext->shareNgGetLastCreatedLinkShareToken()
		: $this->featureContext->getLastCreatedPublicShareToken();
		$properties = null;
		foreach ($propertiesTable->getRows() as $row) {
			$properties[] = $row[0];
		}
		return $this->featureContext->listFolder(
			$token,
			$path,
			'0',
			$properties,
			null,
			"public-files"
		);
	}

	/**
	 *
	 * @param string $path
	 * @param TableNode $propertiesTable
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('/^the public gets the following properties of (?:file|folder|entry) "([^"]*)" in the last created public link using the WebDAV API$/')]
	public function thePublicGetsFollowingPropertiesOfEntryFromLastLinkShare(
		string $path,
		TableNode $propertiesTable
	): void {
		$response = $this->getPropertiesOfEntryFromLastLinkShare($path, $propertiesTable);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @param string $user user id who sets the property
	 * @param string $propertyName name of property in Clark notation
	 * @param string $path path on which to set properties to
	 * @param string $propertyValue property value
	 * @param string|null $namespace namespace in the form of "x1='http://whatever.org/ns'"
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function setResourceProperty(
		string $user,
		string $propertyName,
		string $path,
		string $propertyValue,
		?string $namespace = null,
	): ResponseInterface {
		$user = $this->featureContext->getActualUsername($user);
		return WebDavHelper::proppatch(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getUserPassword($user),
			$path,
			$propertyName,
			$propertyValue,
			$this->featureContext->getStepLineRef(),
			$namespace,
			$this->featureContext->getDavPathVersion()
		);
	}

	/**
	 *
	 * @param string $user user id who sets the property
	 * @param string $propertyName name of property in Clark notation
	 * @param string $path path on which to set properties to
	 * @param string $propertyValue property value
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Given('/^user "([^"]*)" sets property "([^"]*)" of (?:file|folder|entry) "([^"]*)" to "([^"]*)"$/')]
	public function userSetsPropertyOfEntryTo(
		string $user,
		string $propertyName,
		string $path,
		string $propertyValue
	): void {
		$response = $this->setResourceProperty(
			$user,
			$propertyName,
			$path,
			$propertyValue
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $user user id who sets the property
	 * @param string $propertyName name of property in Clark notation
	 * @param string $namespace namespace in the form of "x1='http://whatever.org/ns'"
	 * @param string $path path on which to set properties to
	 * @param string $propertyValue property value
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('/^user "([^"]*)" sets property "([^"]*)" with namespace "([^"]*)" of (?:file|folder|entry) "([^"]*)" to "([^"]*)" using the WebDAV API$/')]
	public function userSetsPropertyWithNamespaceOfEntryTo(
		string $user,
		string $propertyName,
		string $namespace,
		string $path,
		string $propertyValue
	): void {
		$response = $this->setResourceProperty(
			$user,
			$propertyName,
			$path,
			$propertyValue,
			$namespace
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $user user id who sets the property
	 * @param string $propertyName name of property in Clark notation
	 * @param string $path path on which to set properties to
	 * @param string $propertyValue property value
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Given('/^user "([^"]*)" has set property "([^"]*)" of (?:file|folder|entry) "([^"]*)" to "([^"]*)"$/')]
	public function userHasSetPropertyOfEntryTo(
		string $user,
		string $propertyName,
		string $path,
		string $propertyValue
	): void {
		$response = $this->setResourceProperty(
			$user,
			$propertyName,
			$path,
			$propertyValue
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(207, "", $response);
	}

	/**
	 *
	 * @param string $propertyName
	 * @param string $propertyValue
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('/^the response should contain a custom "([^"]*)" property with value "(([^"\\\\]|\\\\.)*)"$/')]
	public function theResponseShouldContainCustomPropertyWithValue(string $propertyName, string $propertyValue): void {
		$propertyValue = \str_replace('\"', '"', $propertyValue);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$this->featureContext->getResponse(),
			__METHOD__
		);
		$xmlPart = $responseXmlObject->xpath(
			"//d:prop/" . "$propertyName"
		);
		Assert::assertArrayHasKey(
			0,
			$xmlPart,
			"Cannot find property \"$propertyName\""
		);
		Assert::assertEquals(
			$propertyValue,
			$xmlPart[0]->__toString(),
			"\"$propertyName\" has a value \"" .
			$xmlPart[0]->__toString() . "\" but \"$propertyValue\" expected"
		);
	}

	/**
	 *
	 * @param string $propertyName
	 * @param string $namespaceString
	 * @param string $propertyValue
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the response should contain a custom :propertyName property with namespace :namespaceString and value :propertyValue')]
	public function theResponseShouldContainACustomPropertyWithNamespaceAndValue(
		string $propertyName,
		string $namespaceString,
		string $propertyValue
	): void {
		// let's unescape quotes first
		$propertyValue = \str_replace('\"', '"', $propertyValue);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$this->featureContext->getResponse(),
			__METHOD__
		);
		$ns = WebDavHelper::parseNamespace($namespaceString);
		$responseXmlObject->registerXPathNamespace(
			$ns->prefix,
			$ns->namespace
		);
		$xmlPart = $responseXmlObject->xpath(
			"//d:prop/$propertyName"
		);
		Assert::assertArrayHasKey(
			0,
			$xmlPart,
			"Cannot find property \"$propertyName\""
		);
		Assert::assertEquals(
			$propertyValue,
			$xmlPart[0]->__toString(),
			"\"$propertyName\" has a value \"" .
			$xmlPart[0]->__toString() . "\" but \"$propertyValue\" expected"
		);
	}

	/**
	 *
	 * @param string $property
	 * @param string $childProperty
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	#[Then('the single response should contain a property :property with a child property :childProperty')]
	public function theSingleResponseShouldContainAPropertyWithChildProperty(
		string $property,
		string $childProperty
	): void {
		$xmlPart = HttpRequestHelper::getResponseXml($this->featureContext->getResponse())->xpath(
			"//d:prop/$property/$childProperty"
		);
		Assert::assertTrue(
			isset($xmlPart[0]),
			"Cannot find property \"$property/$childProperty\""
		);
	}

	/**
	 *
	 * @param string $key
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the xml response should contain a property :key')]
	public function theResponseShouldContainProperty(string $key): void {
		$this->checkResponseContainsProperty(
			$this->featureContext->getResponse(),
			$key
		);
	}

	/**
	 *
	 * @param string $key
	 * @param string $namespace
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the xml response should contain a property :key with namespace :namespace')]
	public function theResponseShouldContainPropertyWithNamespace(string $key, string $namespace): void {
		$this->checkResponseContainsProperty(
			$this->featureContext->getResponse(),
			$key,
			$namespace
		);
	}

	/**
	 *
	 * @param string $key
	 * @param string $expectedValue
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the single response should contain a property :key with value :value')]
	public function theSingleResponseShouldContainAPropertyWithValue(
		string $key,
		string $expectedValue
	): void {
		$this->checkResponseContainsAPropertyWithValue(
			$this->featureContext->getResponse(),
			$key,
			$expectedValue,
			$expectedValue
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $key
	 * @param string $expectedValue
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the single response about the file owned by :user should contain a property :key with value :value')]
	public function theSingleResponseAboutTheFileOwnedByShouldContainAPropertyWithValue(
		string $user,
		string $key,
		string $expectedValue
	): void {
		$this->checkResponseContainsAPropertyWithValue(
			$this->featureContext->getResponse(),
			$key,
			$expectedValue,
			$expectedValue,
			$user
		);
	}

	/**
	 * @param ResponseInterface $response
	 * @param string $key
	 * @param string|null $namespaceString
	 *
	 * @return SimpleXMLElement
	 * @throws Exception
	 */
	public function checkResponseContainsProperty(
		ResponseInterface $response,
		string $key,
		?string $namespaceString = null
	): SimpleXMLElement {
		$xmlPart = HttpRequestHelper::getResponseXml($response, __METHOD__);
		;

		if ($namespaceString !== null) {
			$ns = WebDavHelper::parseNamespace($namespaceString);
			$xmlPart->registerXPathNamespace(
				$ns->prefix,
				$ns->namespace
			);
		}

		$match = $xmlPart->xpath("//d:prop/$key");

		Assert::assertTrue(
			isset($match[0]),
			"Cannot find property \"$key\""
		);

		$property = \explode(":", $key);
		$propertyName = $property[\count($property) - 1];
		Assert::assertEquals(
			$match[0]->getName(),
			$propertyName
		);
		return $match[0];
	}

	/**
	 * @param ResponseInterface $response
	 * @param string $key
	 * @param string $expectedValue
	 * @param string $altExpectedValue
	 * @param string|null $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkResponseContainsAPropertyWithValue(
		ResponseInterface $response,
		string $key,
		string $expectedValue,
		string $altExpectedValue,
		?string $user = null
	): void {
		$xmlPart = $this->checkResponseContainsProperty($response, $key);
		$value = $xmlPart->__toString();
		$expectedValue = $this->featureContext->substituteInLineCodes(
			$expectedValue,
			$user
		);
		$expectedValue = "#^$expectedValue$#";
		$altExpectedValue = "#^$altExpectedValue$#";
		if (\preg_match($expectedValue, $value) !== 1
			&& \preg_match($altExpectedValue, $value) !== 1
		) {
			Assert::fail(
				"Property \"$key\" found with value \"$value\", " .
				"expected \"$expectedValue\" or \"$altExpectedValue\""
			);
		}
	}

	/**
	 *
	 * @param string $xpath
	 * @param string $expectedValue
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the value of the item :xpath in the response should be :value')]
	public function assertValueOfItemInResponseIs(string $xpath, string $expectedValue): void {
		$this->assertValueOfItemInResponseAboutUserIs(
			$xpath,
			null,
			$expectedValue
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $xpath
	 * @param string $path
	 * @param string $expectedValue
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('as user :user the value of the item :xpath of path :path in the response should be :value')]
	public function valueOfItemOfPathShouldBe(string $user, string $xpath, string $path, string $expectedValue): void {
		$path = $this->featureContext->substituteInLineCodes($path, $user);
		$path = \ltrim($path, '/');
		$path = "/" . WebdavHelper::prefixRemotePhp($path);
		$fullXpath = "//d:response/d:href[.='$path']/following-sibling::d:propstat$xpath";
		$this->assertValueOfItemInResponseAboutUserIs(
			$fullXpath,
			null,
			$expectedValue
		);
	}

	/**
	 *
	 * @param string $xpath
	 * @param string $user
	 * @param string $expectedValue
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the value of the item :xpath in the response about user :user should be :value')]
	public function theValueOfTheItemInTheResponseAboutUserShouldBe(
		string $xpath,
		string $user,
		string $expectedValue
	): void {
		$this->assertValueOfItemInResponseAboutUserIs($xpath, $user, $expectedValue);
	}

	/**
	 * @param string $xpath
	 * @param string|null $user
	 * @param string $expectedValue
	 *
	 * @return void
	 */
	public function assertValueOfItemInResponseAboutUserIs(string $xpath, ?string $user, string $expectedValue): void {
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$this->featureContext->getResponse(),
			__METHOD__
		);
		$value = $this->getXmlItemByXpath($responseXmlObject, $xpath);
		$user = $this->featureContext->getActualUsername($user);
		$expectedValue = $this->featureContext->substituteInLineCodes(
			$expectedValue,
			$user
		);

		// The expected value can contain /%base_path%/ which can be empty some time
		// This will result in urls starting from '//', so replace that with single'/'
		$expectedValue = preg_replace("/^\/\//i", "/", $expectedValue);
		Assert::assertEquals(
			$expectedValue,
			$value,
			"item \"$xpath\" found with value \"$value\", " .
			"expected \"$expectedValue\""
		);
	}

	/**
	 *
	 * @param string $xpath
	 * @param string|null $user
	 * @param string $expectedValue1
	 * @param string $expectedValue2
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the value of the item :xpath in the response about user :user should be :value1 or :value2')]
	public function assertValueOfItemInResponseAboutUserIsEitherOr(
		string $xpath,
		?string $user,
		string $expectedValue1,
		string $expectedValue2
	): void {
		if (!$expectedValue2) {
			$expectedValue2 = $expectedValue1;
		}
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$this->featureContext->getResponse(),
			__METHOD__
		);
		$value = $this->getXmlItemByXpath($responseXmlObject, $xpath);
		$user = $this->featureContext->getActualUsername($user);
		$expectedValue1 = $this->featureContext->substituteInLineCodes(
			$expectedValue1,
			$user
		);

		$expectedValue2 = $this->featureContext->substituteInLineCodes(
			$expectedValue2,
			$user
		);

		// The expected value can contain /%base_path%/ which can be empty some time
		// This will result in urls starting from '//', so replace that with single'/'
		$expectedValue1 = preg_replace("/^\/\//i", "/", $expectedValue1);
		$expectedValue2 = preg_replace("/^\/\//i", "/", $expectedValue2);
		$expectedValues = [$expectedValue1, $expectedValue2];
		$isExpectedValueInMessage = \in_array($value, $expectedValues);
		Assert::assertTrue(
			$isExpectedValueInMessage,
			"The actual value \"$value\" is not one of the expected values: \"$expectedValue1\" or \"$expectedValue2\""
		);
	}

	/**
	 * @param SimpleXMLElement $responseXmlObject
	 * @param string $xpath
	 *
	 * @return string
	 */
	public function getXmlItemByXpath(
		SimpleXMLElement $responseXmlObject,
		string $xpath
	): string {
		$xmlPart = $responseXmlObject->xpath($xpath);
		Assert::assertTrue(
			isset($xmlPart[0]),
			"Cannot find item with xpath \"$xpath\""
		);
		return $xmlPart[0]->__toString();
	}

	/**
	 *
	 * @param string $xpath
	 * @param string $pattern
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the value of the item :xpath in the response should match :value')]
	public function assertValueOfItemInResponseRegExp(string $xpath, string $pattern): void {
		$this->assertXpathValueMatchesPattern(
			HttpRequestHelper::getResponseXml($this->featureContext->getResponse()),
			$xpath,
			$pattern
		);
	}

	/**
	 *
	 * @param string $expectedHref
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('there should be an entry with href containing :expectedHref in the response to user :user')]
	public function assertEntryWithHrefMatchingRegExpInResponseToUser(string $expectedHref, string $user): void {
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$this->featureContext->getResponse(),
			__METHOD__
		);

		$user = $this->featureContext->getActualUsername($user);
		$expectedHref = $this->featureContext->substituteInLineCodes(
			$expectedHref,
			$user,
			['preg_quote' => ['/']]
		);
		$expectedHref = WebdavHelper::prefixRemotePhp($expectedHref);

		$index = 0;
		while (true) {
			$index++;
			$xpath = "//d:response[$index]/d:href";
			$xmlPart = $responseXmlObject->xpath($xpath);
			// If we have run out of entries in the response, then fail the test
			Assert::assertTrue(
				isset($xmlPart[0]),
				"Cannot find any entry having href with value $expectedHref in response to $user"
			);
			$value = $xmlPart[0]->__toString();
			$decodedValue = \rawurldecode($value);
			$explodeDecoded = \explode('/', $decodedValue);
			// get the first item of the expected href.
			// 'dav' from "dav/spaces/%spaceid%/C++ file.cpp"
			$explodeExpected = \explode('/', $expectedHref)[0];

			$remotePhpIndex = \array_search($explodeExpected, $explodeDecoded);
			if ($remotePhpIndex) {
				$explodedHrefPartArray = \array_slice($explodeDecoded, $remotePhpIndex);
				$actualHrefPart = \implode('/', $explodedHrefPartArray);
				if ($this->featureContext->getDavPathVersion() === WebDavHelper::DAV_VERSION_SPACES) {
					// for spaces webdav, space id is included in the href
					// space id from our helper is returned as d8c029e0\-2bc9\-4b9a\-8613\-c727e5417f05
					// so we've to remove "\" before every "-"
					$expectedHref = str_replace('\-', '-', $expectedHref);
					$expectedHref = str_replace('\$', '$', $expectedHref);
					$expectedHref = str_replace('\!', '!', $expectedHref);
				}
				if ($actualHrefPart === $expectedHref) {
					break;
				}
			}
		}
	}

	/**
	 * @param SimpleXMLElement $responseXmlObject
	 * @param string $xpath
	 * @param string $pattern
	 * @param string|null $user
	 *
	 * @return void
	 * @throws JsonException
	 */
	public function assertXpathValueMatchesPattern(
		SimpleXMLElement $responseXmlObject,
		string $xpath,
		string $pattern,
		?string $user = null
	): void {
		$xmlPart = $responseXmlObject->xpath($xpath);
		Assert::assertTrue(
			isset($xmlPart[0]),
			"Cannot find item with xpath \"$xpath\""
		);
		$user = $this->featureContext->getActualUsername($user);
		$value = $xmlPart[0]->__toString();
		$callback = ($this->featureContext->isUsingSharingNG())
		? "shareNgGetLastCreatedLinkShareToken" : "getLastCreatedPublicShareToken";

		if (\str_ends_with($xpath, "d:href")) {
			$pattern = \preg_replace("/^\//", "", $pattern);
			$pattern = \preg_replace("/^\^/", "", $pattern);
			$pattern = \ltrim($pattern, "\/");
			$prefixRemotePhp = \rtrim(WebdavHelper::prefixRemotePhp(""), "/");
			$pattern = "/^\/{$prefixRemotePhp}\/{$pattern}";
		}
		$pattern = $this->featureContext->substituteInLineCodes(
			$pattern,
			$user,
			['preg_quote' => ['/']],
			[
				[
					"code" => "%public_token%",
					"function" =>
					[$this->featureContext, $callback],
					"parameter" => []
				],
			]
		);
		Assert::assertMatchesRegularExpression(
			$pattern,
			$value,
			"item \"$xpath\" found with value \"$value\", " .
			"expected to match regex pattern: \"$pattern\""
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $property
	 * @param string $expectedValue
	 * @param string|null $altExpectedValue
	 *
	 * @return void
	 */
	#[Then('/^as user "([^"]*)" (?:file|folder|entry) "([^"]*)" should contain a property "([^"]*)" with value "([^"]*)" or with value "([^"]*)"$/')]
	#[Then('/^as user "([^"]*)" (?:file|folder|entry) "([^"]*)" should contain a property "([^"]*)" with value "([^"]*)"$/')]
	public function asUserFolderShouldContainAPropertyWithValueOrWithValue(
		string $user,
		string $path,
		string $property,
		string $expectedValue,
		?string $altExpectedValue = null
	): void {
		$this->checkPropertyOfAFolder($user, $path, $property, $expectedValue, $altExpectedValue);
	}

	/**
	 * @param string $user
	 * @param string $path
	 * @param string $property
	 * @param string $expectedValue
	 * @param string|null $altExpectedValue
	 * @param string|null $spaceId
	 *
	 * @return void
	 */
	public function checkPropertyOfAFolder(
		string $user,
		string $path,
		string $property,
		string $expectedValue,
		?string $altExpectedValue = null,
		?string $spaceId = null,
	): void {
		$response = $this->featureContext->listFolder(
			$user,
			$path,
			'0',
			[$property],
			$spaceId,
		);
		if ($altExpectedValue === null) {
			$altExpectedValue = $expectedValue;
		}
		$this->checkResponseContainsAPropertyWithValue(
			$response,
			$property,
			$expectedValue,
			$altExpectedValue
		);
	}

	/**
	 *
	 * @param string $key
	 * @param string $regex
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the single response should contain a property :key with value like :regex')]
	public function theSingleResponseShouldContainAPropertyWithValueLike(
		string $key,
		string $regex
	): void {
		$xmlPart = HttpRequestHelper::getResponseXml($this->featureContext->getResponse())->xpath(
			"//d:prop/$key"
		);
		Assert::assertTrue(
			isset($xmlPart[0]),
			"Cannot find property \"$key\""
		);
		$value = $xmlPart[0]->__toString();
		Assert::assertMatchesRegularExpression(
			$regex,
			$value,
			"Property \"$key\" found with value \"$value\", expected \"$regex\""
		);
	}

	/**
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the response should contain a share-types property with')]
	public function theResponseShouldContainAShareTypesPropertyWith(TableNode $table): void {
		$this->featureContext->verifyTableNodeColumnsCount($table, 1);
		WebdavTest::assertResponseContainsShareTypes(
			HttpRequestHelper::getResponseXml($this->featureContext->getResponse()),
			$table->getRows()
		);
	}

	/**
	 *
	 * @param string $property
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the response should contain an empty property :property')]
	public function theResponseShouldContainAnEmptyProperty(string $property): void {
		$xmlPart = HttpRequestHelper::getResponseXml($this->featureContext->getResponse())->xpath(
			"//d:prop/$property"
		);
		Assert::assertCount(
			1,
			$xmlPart,
			"Cannot find property \"$property\""
		);
		Assert::assertEmpty(
			$xmlPart[0],
			"Property \"$property\" is not empty"
		);
	}

	/**
	 * @param string $user
	 * @param string $path
	 * @param string|null $storePath
	 * @param string|null $spaceId
	 *
	 * @return SimpleXMLElement
	 * @throws Exception
	 */
	public function storeEtagOfElement(
		string $user,
		string $path,
		?string $storePath = "",
		?string  $spaceId = null
	): SimpleXMLElement {
		if ($storePath === "") {
			$storePath = $path;
		}
		$user = $this->featureContext->getActualUsername($user);
		$propertiesTable = new TableNode([['propertyName'],['d:getetag']]);
		$response = $this->getPropertiesOfFolder(
			$user,
			$path,
			$spaceId,
			$propertiesTable
		);
		$xmlObject = HttpRequestHelper::getResponseXml($response, __METHOD__);
		$this->storedETAG[$user][$storePath]
			= $this->featureContext->getEtagFromResponseXmlObject($xmlObject);
		return $xmlObject;
	}

	/**
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $storePath
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Given('user :user has stored etag of element :path on path :storePath')]
	public function userStoresEtagOfElementOnPath(string $user, string $path, string $storePath): void {
		$user = $this->featureContext->getActualUsername($user);
		$this->storeEtagOfElement(
			$user,
			$path,
			$storePath
		);
		if ($storePath == "") {
			$storePath = $path;
		}
		if ($this->storedETAG[$user][$storePath] === null || $this->storedETAG[$user][$path] === "") {
			throw new Exception("Expected stored etag to be some string but found null!");
		}
	}

	/**
	 *
	 * @param string $user
	 * @param string $path
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Given('user :user has stored etag of element :path')]
	public function userHasStoredEtagOfElement(string $user, string $path): void {
		$user = $this->featureContext->getActualUsername($user);
		$this->storeEtagOfElement(
			$user,
			$path
		);
		if ($this->storedETAG[$user][$path] === "" || $this->storedETAG[$user][$path] === null) {
			throw new Exception("Expected stored etag to be some string but found null!");
		}
	}

	/**
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the properties response should contain an etag')]
	public function thePropertiesResponseShouldContainAnEtag(): void {
		Assert::assertTrue(
			$this->featureContext->isEtagValid($this->featureContext->getEtagFromResponseXmlObject()),
			__METHOD__
			. " getetag not found in response"
		);
	}

	/**
	 * @param string $href
	 *
	 * @return string
	 */
	public function parseBaseDavPathFromXMLHref(string $href): string {
		$hrefArr = \explode('/', $href);
		if (\in_array("webdav", $hrefArr)) {
			$hrefArr = \array_slice($hrefArr, 0, \array_search("webdav", $hrefArr) + 1);
		} elseif (\in_array("files", $hrefArr)) {
			$hrefArr = \array_slice($hrefArr, 0, \array_search("files", $hrefArr) + 2);
		} elseif (\in_array("spaces", $hrefArr)) {
			$hrefArr = \array_slice($hrefArr, 0, \array_search("spaces", $hrefArr) + 2);
		}
		return \implode('/', $hrefArr);
	}

	/**
	 * only supports new DAV version
	 *
	 * @param string $username
	 * @param TableNode $expectedPropTable with the following columns:
	 *                                     resource: full path of resource(file/folder/entry) from root of your oc storage
	 *                                     property: expected name of property to be asserted, e.g.: status, href, customPropName
	 *                                     value: expected value of expected property
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('as user :username the last response should have the following properties')]
	public function theResponseShouldHavePropertyWithValue(string $username, TableNode $expectedPropTable): void {
		$this->featureContext->verifyTableNodeColumns(
			$expectedPropTable,
			['resource', 'propertyName', 'propertyValue']
		);
		$responseXmlObject = HttpRequestHelper::getResponseXml($this->featureContext->getResponse());

		$href = (string)$responseXmlObject->xpath("//d:href")[0];
		$hrefBase = $this->parseBaseDavPathFromXMLHref($href);

		foreach ($expectedPropTable->getColumnsHash() as $col) {
			$xpath = "//d:href[.='$hrefBase" . $col["resource"] . "']" .
				"/following-sibling::d:propstat//" . $col["propertyName"];
			$xmlPart = $responseXmlObject->xpath($xpath);

			Assert::assertEquals(
				$col["propertyValue"],
				$xmlPart[0],
				__METHOD__
				. " Expected '" . $col["propertyValue"] . "' but got '" . $xmlPart[0] . "'"
			);
		}
	}

	/**
	 * @param string $path
	 * @param string $user
	 *
	 * @return string
	 * @throws Exception
	 */
	public function getCurrentEtagOfElement(string $path, string $user): string {
		$user = $this->featureContext->getActualUsername($user);
		$propertiesTable = new TableNode([['propertyName'],['d:getetag']]);
		$response = $this->getPropertiesOfFolder(
			$user,
			$path,
			null,
			$propertiesTable
		);
		return $this->featureContext->getEtagFromResponseXmlObject(
			HttpRequestHelper::getResponseXml($response, __METHOD__)
		);
	}

	/**
	 * @param string $path
	 * @param string $user
	 * @param string $messageStart
	 *
	 * @return string
	 */
	public function getStoredEtagOfElement(string $path, string $user, string $messageStart = ''): string {
		if ($messageStart === '') {
			$messageStart = __METHOD__;
		}
		Assert::assertArrayHasKey(
			$user,
			$this->storedETAG,
			$messageStart
			. " Trying to check etag of element $path of user $user but the user does not have any stored etags"
		);
		Assert::assertArrayHasKey(
			$path,
			$this->storedETAG[$user],
			$messageStart
			. " Trying to check etag of element $path of user "
			. "$user but the user does not have a stored etag for the element"
		);
		return $this->storedETAG[$user][$path];
	}

	/**
	 *
	 * @param TableNode $etagTable
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('these etags should not have changed:')]
	public function theseEtagsShouldNotHaveChanged(TableNode $etagTable): void {
		$this->featureContext->verifyTableNodeColumns($etagTable, ["user", "path"]);
		$this->featureContext->verifyTableNodeColumnsCount($etagTable, 2);
		$changedEtagCount = 0;
		$changedEtagMessage = __METHOD__;
		foreach ($etagTable->getColumnsHash() as $row) {
			$user = $row["user"];
			$path = $row["path"];
			$user = $this->featureContext->getActualUsername($user);
			$actualEtag = $this->getCurrentEtagOfElement($path, $user);
			$storedEtag = $this->getStoredEtagOfElement($path, $user, __METHOD__);
			if ($actualEtag !== $storedEtag) {
				$changedEtagCount = $changedEtagCount + 1;
				$changedEtagMessage
					.= "\nThe etag '$storedEtag' of element '$path' of user '$user' changed to '$actualEtag'.";
			}
		}
		Assert::assertEquals(0, $changedEtagCount, $changedEtagMessage);
	}

	/**
	 *
	 * @param TableNode $etagTable
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('these etags should have changed:')]
	public function theseEtagsShouldHaveChanged(TableNode $etagTable): void {
		$this->featureContext->verifyTableNodeColumns($etagTable, ["user", "path"]);
		$this->featureContext->verifyTableNodeColumnsCount($etagTable, 2);
		$unchangedEtagCount = 0;
		$unchangedEtagMessage = __METHOD__;
		foreach ($etagTable->getColumnsHash() as $row) {
			$user = $row["user"];
			$path = $row["path"];
			$user = $this->featureContext->getActualUsername($user);
			$actualEtag = $this->getCurrentEtagOfElement($path, $user);
			$storedEtag = $this->getStoredEtagOfElement($path, $user, __METHOD__);
			if ($actualEtag === $storedEtag) {
				$unchangedEtagCount = $unchangedEtagCount + 1;
				$unchangedEtagMessage
					.= "\nThe etag '$storedEtag' of element '$path' of user '$user' did not change.";
			}
		}
		Assert::assertEquals(0, $unchangedEtagCount, $unchangedEtagMessage);
	}

	/**
	 *
	 * @param string $path
	 * @param string $user
	 * @param string $shouldShouldNot
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('/^the etag of element "([^"]*)" of user "([^"]*)" (should|should not) have changed$/')]
	public function etagOfElementOfUserShouldOrShouldNotHaveChanged(
		string $path,
		string $user,
		string $shouldShouldNot
	): void {
		$user = $this->featureContext->getActualUsername($user);
		$actualEtag = $this->getCurrentEtagOfElement($path, $user);
		$storedEtag = $this->getStoredEtagOfElement($path, $user, __METHOD__);
		if ($shouldShouldNot === 'should not') {
			Assert::assertEquals(
				$storedEtag,
				$actualEtag,
				__METHOD__
				. " The etag of element '$path' of user '$user' was not expected to change."
				. " The stored etag was '$storedEtag' but got '$actualEtag' from the response"
			);
		} else {
			Assert::assertNotEquals(
				$storedEtag,
				$actualEtag,
				__METHOD__
				. " The etag of element '$path' of user '$user' was expected to change."
				. " The stored etag was '$storedEtag' and also got '$actualEtag' from the response"
			);
		}
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
