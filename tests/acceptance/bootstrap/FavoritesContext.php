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
use TestHelpers\BehatHelper;

require_once 'bootstrap.php';

/**
 * context containing favorites related API steps
 */
class FavoritesContext implements Context {
	private WebDavPropertiesContext $webDavPropertiesContext;

	/**
	 * @Then /^as user "([^"]*)" (?:file|folder|entry) "([^"]*)" should be favorited$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param integer $expectedValue 0|1
	 * @param string|null $spaceId
	 *
	 * @return void
	 */
	public function asUserFileOrFolderShouldBeFavorited(
		string $user,
		string $path,
		int $expectedValue = 1,
		?string $spaceId = null
	): void {
		$property = "oc:favorite";
		$this->webDavPropertiesContext->checkPropertyOfAFolder(
			$user,
			$path,
			$property,
			(string)$expectedValue,
			null,
			$spaceId,
		);
	}

	/**
	 * @Then /^as user "([^"]*)" (?:file|folder|entry) "([^"]*)" should not be favorited$/
	 *
	 * @param string $user
	 * @param string $path
	 *
	 * @return void
	 */
	public function asUserFileShouldNotBeFavorited(string $user, string $path): void {
		$this->asUserFileOrFolderShouldBeFavorited($user, $path, 0);
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
		$this->webDavPropertiesContext = BehatHelper::getContext(
			$scope,
			$environment,
			'WebDavPropertiesContext'
		);
	}
}
