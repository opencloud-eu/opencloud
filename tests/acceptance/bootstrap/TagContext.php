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
		// Note: after creating or deleting tags, in some cases tags do not appear or disappear immediately,
		// So wait is necessary before listing tags
		sleep(5);
		$this->featureContext->setResponse(
			GraphHelper::getTags(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 *
	 * @param string    $shouldOrNot   (not|)
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('/^the response should (not|)\\s?contain following tag(s):$/')]
	public function theFollowingTagsShouldExistForUser(string $shouldOrNot, TableNode $table): void {
		$rows = $table->getRows();
		foreach ($rows as $row) {
			$responseArray = $this->featureContext->getJsonDecodedResponse(
				$this->featureContext->getResponse()
			)['value'];
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
