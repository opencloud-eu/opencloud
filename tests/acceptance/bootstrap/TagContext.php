<?php declare(strict_types=1);
/**
 * @author Viktor Scharf <scharf.vi@gmail.com>
 *
 * @copyright Copyright (c) 2022, ownCloud GmbH
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
use Behat\Gherkin\Node\TableNode;
use PHPUnit\Framework\Assert;
use Psr\Http\Message\ResponseInterface;
use TestHelpers\GraphHelper;
use TestHelpers\WaitHelper;
use TestHelpers\BehatHelper;
use Behat\Step\Given;
use Behat\Step\Then;
use Behat\Step\When;

require_once 'bootstrap.php';

/**
 * Acceptance test steps related to testing tags features
 */
class TagContext implements Context {
	private FeatureContext $featureContext;
	private SpacesContext $spacesContext;
	private array $lastTagsQuery = [];

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
	 * @param string $user
	 * @param string $fileOrFolder   (file|folder)
	 * @param string $resource
	 * @param string $space
	 * @param TableNode $table
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function createTags(
		string $user,
		string $fileOrFolder,
		string $resource,
		string $space,
		TableNode $table
	): ResponseInterface {
		$tagNameArray = [];
		foreach ($table->getRows() as $value) {
			$tagNameArray[] = $value[0];
		}
		if ($fileOrFolder === 'folder' || $fileOrFolder === 'folders') {
			$resourceId = $this->spacesContext->getResourceId($user, $space, $resource);
		} else {
			$resourceId = $this->spacesContext->getFileId($user, $space, $resource);
		}

		return GraphHelper::createTags(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$resourceId,
			$tagNameArray
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $fileOrFolder   (file|folder)
	 * @param string $resource
	 * @param string $space
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('/^user "([^"]*)" creates the following tags for (folder|file) "([^"]*)" of space "([^"]*)":$/')]
	public function theUserCreatesFollowingTags(
		string $user,
		string $fileOrFolder,
		string $resource,
		string $space,
		TableNode $table
	): void {
		$response = $this->createTags($user, $fileOrFolder, $resource, $space, $table);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $fileOrFolder   (file|folder)
	 * @param string $resource
	 * @param string $space
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Given('/^user "([^"]*)" has created the following tags for (folder|file)\\s?"([^"]*)" of the space "([^"]*)":$/')]
	public function theUserHasCreatedFollowingTags(
		string $user,
		string $fileOrFolder,
		string $resource,
		string $space,
		TableNode $table
	): void {
		$response = $this->createTags($user, $fileOrFolder, $resource, $space, $table);
		$this->featureContext->theHttpStatusCodeShouldBe(200, "", $response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $filesOrFolders (files|folders)
	 * @param string $space
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Given('/^user "([^"]*)" has tagged the following (folders|files) of the space "([^"]*)":$/')]
	public function userHasCreatedTheFollowingTagsForFilesOfTheSpace(
		string $user,
		string $filesOrFolders,
		string $space,
		TableNode $table
	): void {
		$this->featureContext->verifyTableNodeColumns($table, ["path", "tagName"]);
		$rows = $table->getHash();
		foreach ($rows as $row) {
			$tags = explode(',', $row['tagName']);
			$response = $this->createTags($user, $filesOrFolders, $row['path'], $space, new TableNode([$tags]));
			$this->featureContext->theHttpStatusCodeShouldBe(200, "", $response);
		}
	}

	/**
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('user :user lists all available tag(s) via the Graph API')]
	public function theUserGetsAllAvailableTags(string $user): void {
		// Note: after creating or deleting tags, in some cases tags do not appear or disappear immediately
		sleep(2);
		$this->lastTagsQuery = ["user" => $user];
		$this->featureContext->setResponse($this->fetchTags($user));
	}

	/**
	 * @param string $user
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	private function fetchTags(string $user): ResponseInterface {
		return GraphHelper::getTags(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$this->featureContext->getStepLineRef()
		);
	}

	/**
	 * re-run the last tag listing until the assertion passes or the WaitHelper
	 *
	 * @param callable $assert
	 *
	 * @return void
	 */
	private function retryTagsUntilSatisfied(callable $assert): void {
		Assert::assertNotEmpty(
			$this->lastTagsQuery,
			'No tag listing to retry. Use a "lists all available tags via the Graph API" step first.'
		);
		$query = $this->lastTagsQuery;
		$response = WaitHelper::waitUntil(
			fn () => $this->fetchTags($query["user"]),
			function ($response) use ($assert) {
				$this->featureContext->setResponse($response);
				try {
					$assert();
					return true;
				} catch (\Throwable) {
					return false;
				}
			}
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('/^the response should contain following tags:$/')]
	public function theResponseShouldContainFollowingTags(TableNode $table): void {
		$this->assertResponseContainsFollowingTags("", $table);
	}

	/**
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('/^the response should not contain following tags:$/')]
	public function theResponseShouldNotContainFollowingTags(TableNode $table): void {
		$this->assertResponseContainsFollowingTags("not", $table);
	}

	/**
	 *
	 * @param string    $shouldOrNot   (not|)
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	private function assertResponseContainsFollowingTags(string $shouldOrNot, TableNode $table): void {
		$assert = function () use ($shouldOrNot, $table): void {
			$responseArray = $this->featureContext->getJsonDecodedResponse(
				$this->featureContext->getResponse()
			)['value'];
			foreach ($table->getRows() as $row) {
				if ($shouldOrNot === "not") {
					Assert::assertFalse(
						\in_array($row[0], $responseArray),
						"the response should not contain the tag $row[0].\nResponse\n"
						. print_r($responseArray, true)
					);
				} else {
					Assert::assertTrue(
						\in_array($row[0], $responseArray),
						"the response does not contain the tag $row[0].\nResponse\n"
						. print_r($responseArray, true)
					);
				}
			}
		};
		$this->retryTagsUntilSatisfied($assert);
		$assert();
	}

	/**
	 * @param string $user
	 * @param string $fileOrFolder   (file|folder)
	 * @param string $resource
	 * @param string $space
	 * @param TableNode $table
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function removeTagsFromResourceOfTheSpace(
		string $user,
		string $fileOrFolder,
		string $resource,
		string $space,
		TableNode $table
	): ResponseInterface {
		$tagNameArray = [];
		foreach ($table->getRows() as $value) {
			$tagNameArray[] = $value[0];
		}

		if ($fileOrFolder === 'folder') {
			$resourceId = $this->spacesContext->getResourceId($user, $space, $resource);
		} else {
			$resourceId = $this->spacesContext->getFileId($user, $space, $resource);
		}

		return GraphHelper::deleteTags(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$resourceId,
			$tagNameArray
		);
	}

	/**
	 *
	 * @param string $user
	 * @param string $fileOrFolder   (file|folder)
	 * @param string $resource
	 * @param string $space
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[When('/^user "([^"]*)" removes the following tags for (folder|file) "([^"]*)" of space "([^"]*)":$/')]
	public function userRemovesTagsFromResourceOfTheSpace(
		string $user,
		string $fileOrFolder,
		string $resource,
		string $space,
		TableNode $table
	): void {
		$response = $this->removeTagsFromResourceOfTheSpace(
			$user,
			$fileOrFolder,
			$resource,
			$space,
			$table
		);
		$this->featureContext->setResponse($response);
	}

	/**
	 *
	 * @param string $user
	 * @param string $fileOrFolder   (file|folder)
	 * @param string $resource
	 * @param string $space
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Given('/^user "([^"]*)" has removed the following tags for (folder|file) "([^"]*)" of space "([^"]*)":$/')]
	public function userHAsRemovedTheFollowingTagsForFileOfSpace(
		string $user,
		string $fileOrFolder,
		string $resource,
		string $space,
		TableNode $table
	): void {
		$response = $this->removeTagsFromResourceOfTheSpace($user, $fileOrFolder, $resource, $space, $table);
		$this->featureContext->theHttpStatusCodeShouldBe(200, "", $response);
	}
}
