<?php declare(strict_types=1);
/**
 * @author Artur Neumann <artur@jankaritech.com>
 * @copyright Copyright (c) 2018 Artur Neumann artur@jankaritech.com
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
use Psr\Http\Message\ResponseInterface;
use TestHelpers\WebDavHelper;
use TestHelpers\HttpRequestHelper;
use TestHelpers\BehatHelper;

require_once 'bootstrap.php';

/**
 * context containing search related API steps
 */
class SearchContext implements Context {
	private FeatureContext $featureContext;

	/**
	 * @param string $user
	 * @param string $pattern
	 * @param string|null $limit
	 * @param string|null $scopeType
	 * @param string|null $scope
	 * @param string|null $spaceName
	 * @param TableNode|null $properties
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException|JsonException
	 */
	private function searchFiles(
		string     $user,
		string     $pattern,
		?string    $limit = null,
		?string    $scopeType = null,
		?string    $scope = null,
		?string    $spaceName = null,
		?TableNode $properties = null
	): ResponseInterface {
		$user = $this->featureContext->getActualUsername($user);
		$baseUrl = $this->featureContext->getBaseUrl();
		$password = $this->featureContext->getPasswordForUser($user);
		if (str_contains($pattern, '$')) {
			$date = explode("$", $pattern);
			switch ($date[1]) {
				case "today":
					$pattern = $date[0] . date('Y-m-d', strtotime('today'));
					break;
				case "yesterday":
					$pattern = $date[0] . date('Y-m-d', strtotime('yesterday'));
					break;
				default:
					throw new Exception("cannot convert the date");
			}
		}
		$body
			= "<?xml version='1.0' encoding='utf-8' ?>\n" .
			"	<oc:search-files xmlns:a='DAV:' xmlns:oc='http://owncloud.org/ns' >\n" .
			"		<oc:search>\n";
		if ($scope !== null) {
			if ($scopeType === "space") {
				$spaceId = $this->featureContext->spacesContext->getSpaceIdByName($user, $scope);
				$pattern .= " scope:$spaceId";
			} else {
				$resourceID = $this->featureContext->spacesContext->getResourceId(
					$user,
					$spaceName ?? "Personal",
					$scope
				);
				$pattern .= " scope:$resourceID";
			}
		}
		$body .= "<oc:pattern>$pattern</oc:pattern>\n";
		if ($limit !== null) {
			$body .= "			<oc:limit>$limit</oc:limit>\n";
		}

		$body .= "		</oc:search>\n";
		if ($properties !== null) {
			$propertiesRows = $properties->getRows();
			$body .= "	<a:prop>";
			foreach ($propertiesRows as $property) {
				$body .= "<$property[0]/>";
			}
			$body .= "	</a:prop>";
		}
		$body .= "	</oc:search-files>";

		$davPathVersionToUse = $this->featureContext->getDavPathVersion();
		// $davPath will be one of the followings:
		// - webdav
		// - dav/files
		// - dav/spaces
		$davPath = WebDavHelper::getDavPath($davPathVersionToUse);
		$fullUrl = WebDavHelper::sanitizeUrl("$baseUrl/$davPath");

		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			'REPORT',
			$user,
			$password,
			null,
			$body
		);
	}

	/**
	 * @When user :user searches for :pattern using the WebDAV API
	 * @When user :user searches for :pattern and limits the results to :limit items using the WebDAV API
	 * @When user :user searches for :pattern using the WebDAV API requesting these properties:
	 * @When user :user searches for :pattern and limits the results to :limit items using the WebDAV API requesting these properties:
	 *
	 * @param string $user
	 * @param string $pattern
	 * @param string|null $limit
	 * @param TableNode|null $properties
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userSearchesUsingWebDavAPI(
		string     $user,
		string     $pattern,
		?string    $limit = null,
		?TableNode $properties = null
	): void {
		// NOTE: because indexing of newly uploaded files or directories with OpenCloud is decoupled and occurs asynchronously
		// short wait is necessary before searching
		sleep(5);
		$response = $this->searchFiles($user, $pattern, $limit, null, null, null, $properties);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @Then file/folder :path in the search result of user :user should contain these properties:
	 *
	 * @param string $path
	 * @param string $user
	 * @param TableNode $properties
	 *
	 * @return void
	 * @throws Exception
	 */
	public function fileOrFolderInTheSearchResultShouldContainProperties(
		string    $path,
		string    $user,
		TableNode $properties
	): void {
		$user = $this->featureContext->getActualUsername($user);
		$this->featureContext->verifyTableNodeColumns($properties, ['name', 'value']);
		$properties = $properties->getHash();
		$fileResult = $this->featureContext->findEntryFromSearchResponse(
			$path
		);
		Assert::assertNotFalse(
			$fileResult,
			"could not find file/folder '$path'"
		);
		foreach ($properties as $property) {
			$property['value'] = $this->featureContext->substituteInLineCodes(
				$property['value'],
				$user
			);
			if (is_object($fileResult)) {
				$fileResultProperty = $fileResult->xpath("d:propstat//" . $property['name']);
			} else {
				throw new Exception("Expected fileResult to be an object, but found " . gettype($fileResult));
			}
			if ($fileResultProperty) {
				Assert::assertMatchesRegularExpression(
					"/" . $property['value'] . "/",
					\trim((string)$fileResultProperty[0])
				);
				continue;
			}
			throw new Error("Could not find property '" . $property['name'] . "'");
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

	/**
	 * @Then /^the search result should contain these (?:files|entries) with highlight on keyword "([^"]*)"/
	 *
	 * @param TableNode $expectedFiles
	 * @param string $expectedContent
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function theSearchResultShouldContainEntriesWithHighlight(
		TableNode $expectedFiles,
		string    $expectedContent
	): void {
		$this->featureContext->verifyTableNodeColumnsCount($expectedFiles, 1);
		$elementRows = $expectedFiles->getRows();
		$foundEntries = $this->featureContext->findEntryFromSearchResponse(
			null,
			true
		);
		foreach ($elementRows as $expectedFile) {
			$filename = $expectedFile[0];
			$content = $foundEntries[$filename];
			// Extract the content between the <mark> tags
			preg_match('/<mark>(.*?)<\/mark>/s', $content, $matches);
			$actualContent = $matches[1] ?? '';

			// Remove any leading/trailing whitespace for comparison
			$actualContent = trim($actualContent);
			Assert::assertEquals(
				$expectedContent,
				$actualContent,
				"Expected text highlight to be '$expectedContent' but found '$actualContent'"
			);
		}
	}

	/**
	 * @When /^user "([^"]*)" searches for "([^"]*)" inside (folder|space) "([^"]*)" using the WebDAV API$/
	 * @When /^user "([^"]*)" searches for "([^"]*)" inside (folder) "([^"]*)" in space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $pattern
	 * @param string $scopeType
	 * @param string $scope
	 * @param string|null $spaceName
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function userSearchesInsideFolderOrSpaceUsingWebDavAPI(
		string  $user,
		string  $pattern,
		string  $scopeType,
		string  $scope,
		?string $spaceName = null,
	): void {
		// NOTE: since indexing of newly uploaded files or directories with OpenCloud is decoupled and occurs asynchronously,
		// a short wait is necessary before searching
		sleep(5);
		$response = $this-> searchFiles($user, $pattern, null, $scopeType, $scope, $spaceName);
		$this->featureContext->setResponse($response);
	}
}
