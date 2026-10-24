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
use TestHelpers\WaitHelper;
use TestHelpers\HttpRequestHelper;
use TestHelpers\BehatHelper;
use Behat\Step\Then;
use Behat\Step\When;

require_once 'bootstrap.php';

/**
 * context containing search related API steps
 */
class SearchContext implements Context {
	private FeatureContext $featureContext;
	private array $lastSearchQuery = [];

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
	 *
	 * @param string $user
	 * @param string $pattern
	 * @param string|null $limit
	 * @param TableNode|null $properties
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	#[When('user :user searches for :pattern using the WebDAV API')]
	#[When('user :user searches for :pattern and limits the results to :limit items using the WebDAV API')]
	#[When('user :user searches for :pattern using the WebDAV API requesting these properties:')]
	public function userSearchesUsingWebDavAPI(
		string     $user,
		string     $pattern,
		?string    $limit = null,
		?TableNode $properties = null
	): void {
		// NOTE: because indexing of newly uploaded files or directories with OpenCloud is decoupled and occurs asynchronously
		// short wait is necessary before searching
		sleep(2);
		// remember the query so "should eventually contain" steps can re-search
		$this->lastSearchQuery = [
			"user" => $user,
			"pattern" => $pattern,
			"limit" => $limit,
			"properties" => $properties,
		];
		$response = $this->searchFiles($user, $pattern, $limit, null, null, null, $properties);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $path
	 * @param string $user
	 * @param TableNode $properties
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('file/folder :path in the search result of user :user should contain these properties:')]
	public function fileOrFolderInTheSearchResultShouldContainProperties(
		string    $path,
		string    $user,
		TableNode $properties
	): void {
		$assert = fn () => $this->assertFileOrFolderInSearchResultContainsProperties($path, $user, $properties);
		$this->retrySearchUntilSatisfied($assert);
		$assert();
	}

	/**
	 *
	 * @param string $path
	 * @param string $user
	 * @param TableNode $properties
	 *
	 * @return void
	 * @throws Exception
	 */
	private function assertFileOrFolderInSearchResultContainsProperties(
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
			if (\is_object($fileResult)) {
				$fileResultProperty = $fileResult->xpath("d:propstat//" . $property['name']);
			} else {
				throw new Exception("Expected fileResult to be an object, but found " . \gettype($fileResult));
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
	 *
	 * @param TableNode $expectedFiles
	 * @param string $expectedContent
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	private function assertSearchResultContainsEntriesWithHighlight(
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
	#[When('/^user "([^"]*)" searches for "([^"]*)" inside (folder|space) "([^"]*)" using the WebDAV API$/')]
	#[When('/^user "([^"]*)" searches for "([^"]*)" inside (folder) "([^"]*)" in space "([^"]*)" using the WebDAV API$/')]
	public function userSearchesInsideFolderOrSpaceUsingWebDavAPI(
		string  $user,
		string  $pattern,
		string  $scopeType,
		string  $scope,
		?string $spaceName = null,
	): void {
		// NOTE: since indexing of newly uploaded files or directories with OpenCloud is decoupled and occurs asynchronously,
		// a short wait is necessary before searching
		sleep(2);
		$this->lastSearchQuery = [
			"user" => $user,
			"pattern" => $pattern,
			"scopeType" => $scopeType,
			"scope" => $scope,
			"spaceName" => $spaceName,
		];
		$response = $this->searchFiles($user, $pattern, null, $scopeType, $scope, $spaceName);
		$this->featureContext->setResponse($response);
	}

	/**
	 * re-run the last WebDAV search until the assertion passes or the WaitHelper
	 * timeout elapses, leaving the last response set for a final assertion by the
	 * caller. Indexing of newly uploaded resources is asynchronous, so a wanted
	 * file can be missing from an early search; OpenSearch never returns a partial
	 * document, so once the expected entries are present the result is complete.
	 *
	 * @param callable $assert
	 *
	 * @return void
	 */
	public function retrySearchUntilSatisfied(callable $assert): void {
		Assert::assertNotEmpty(
			$this->lastSearchQuery,
			'No search to retry. Use a "searches for ... using the WebDAV API" step first.'
		);
		$query = $this->lastSearchQuery;
		$response = WaitHelper::waitUntil(
			fn () => $this->searchFiles(
				$query["user"],
				$query["pattern"],
				$query["limit"] ?? null,
				$query["scopeType"] ?? null,
				$query["scope"] ?? null,
				$query["spaceName"] ?? null,
				$query["properties"] ?? null
			),
			function ($response) use ($assert) {
				$this->featureContext->setResponse($response);
				try {
					$assert();
					return true;
				} catch (\Throwable) {
					return false;
				}
			},
			2000,
			20
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param int $numFiles
	 *
	 * @return void
	 */
	#[Then('the search result should contain :numFiles files/entries')]
	public function theSearchResultShouldContainNumEntries(int $numFiles): void {
		$assert = fn () => $this->featureContext->checkIFResponseContainsNumberEntries($numFiles);
		$this->retrySearchUntilSatisfied($assert);
		$assert();
	}

	/**
	 *
	 * @param string $user
	 * @param int $expectedNumber
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 */
	#[Then('the search result of user :user should contain any :expectedNumber of these files/entries:')]
	public function theSearchResultShouldContainAnyOfTheseEntries(
		string $user,
		int $expectedNumber,
		TableNode $expectedFiles
	): void {
		$assert = fn () => $this->featureContext->checkIfSearchResultContainsFiles($user, $expectedNumber, $expectedFiles);
		$this->retrySearchUntilSatisfied($assert);
		$assert();
	}

	/**
	 * @param string $user
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 */
	#[Then('/^the search result of user "([^"]*)" should contain only these (?:files|entries):$/')]
	public function theSearchResultShouldContainOnlyEntries(string $user, TableNode $expectedFiles): void {
		$assert = fn () => $this->featureContext->thePropfindResultShouldContainOnlyEntries($user, $expectedFiles);
		$this->retrySearchUntilSatisfied($assert);
		$assert();
	}

	/**
	 * @param string $user
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 */
	#[Then('/^the search result of user "([^"]*)" should contain these (?:files|entries):$/')]
	public function theSearchResultShouldContainEntries(string $user, TableNode $expectedFiles): void {
		$this->assertSearchResultContainsEntries($user, "", $expectedFiles);
	}

	/**
	 * @param string $user
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 */
	#[Then('/^the search result of user "([^"]*)" should not contain these (?:files|entries):$/')]
	public function theSearchResultShouldNotContainEntries(string $user, TableNode $expectedFiles): void {
		$this->assertSearchResultContainsEntries($user, "not", $expectedFiles);
	}

	/**
	 * @param string $user
	 * @param string $shouldOrNot (not|)
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 */
	private function assertSearchResultContainsEntries(string $user, string $shouldOrNot, TableNode $expectedFiles): void {
		$assert = fn () => $this->featureContext->thePropfindResultShouldContainEntries($user, $shouldOrNot, $expectedFiles);
		$this->retrySearchUntilSatisfied($assert);
		$assert();
	}

	/**
	 * @param TableNode $expectedFiles
	 * @param string $expectedContent
	 *
	 * @return void
	 */
	#[Then('/^the search result should contain these (?:files|entries) with highlight on keyword "([^"]*)"$/')]
	public function theSearchResultShouldContainEntriesWithHighlight(
		TableNode $expectedFiles,
		string    $expectedContent
	): void {
		$assert = fn () => $this->assertSearchResultContainsEntriesWithHighlight($expectedFiles, $expectedContent);
		$this->retrySearchUntilSatisfied($assert);
		$assert();
	}
}
