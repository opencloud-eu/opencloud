<?php declare(strict_types=1);
/**
 * @author Sajan Gurung <sajan@jankaritech.com>
 * @copyright Copyright (c) 2023 Sajan Gurung sajan@jankaritech.com
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
use Behat\Gherkin\Node\TableNode;
use GuzzleHttp\Exception\GuzzleException;
use TestHelpers\OcConfigHelper;
use TestHelpers\GraphHelper;
use PHPUnit\Framework\Assert;

/**
 * steps needed to re-configure OpenCloud server
 */
class OcConfigContext implements Context {
	private array $enabledPermissionsRoles = [];

	/**
	 * @return array
	 */
	public function getEnabledPermissionsRoles(): array {
		return $this->enabledPermissionsRoles;
	}

	/**
	 * @param array $enabledPermissionsRoles
	 *
	 * @return void
	 */
	public function setEnabledPermissionsRoles(array $enabledPermissionsRoles): void {
		$this->enabledPermissionsRoles = $enabledPermissionsRoles;
	}

	/**
	 * @Given async upload has been enabled with post-processing delayed to :delayTime seconds
	 *
	 * @param string $delayTime
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function asyncUploadHasBeenEnabledWithDelayedPostProcessing(string $delayTime): void {
		$envs = [
			"OC_ASYNC_UPLOADS" => true,
			"OC_EVENTS_ENABLE_TLS" => false,
			"POSTPROCESSING_DELAY" => $delayTime . "s",
		];

		$response =  OcConfigHelper::reConfigureOc($envs);
		Assert::assertEquals(
			200,
			$response->getStatusCode(),
			"Failed to set async upload with delayed post processing"
		);
		OcConfigHelper::setPostProcessingDelay($delayTime);
	}

	/**
	 * @Given the config :configVariable has been set to :configValue
	 *
	 * @param string $configVariable
	 * @param string $configValue
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function theConfigHasBeenSetTo(string $configVariable, string $configValue): void {
		$envs = [
			$configVariable => $configValue,
		];

		$response =  OcConfigHelper::reConfigureOc($envs);
		Assert::assertEquals(
			200,
			$response->getStatusCode(),
			"Failed to set config $configVariable=$configValue"
		);
		if ($configVariable === "POSTPROCESSING_DELAY") {
			OcConfigHelper::setPostProcessingDelay($configValue);
		}
	}

	/**
	 * @Given the administrator has enabled the permissions role :role
	 *
	 * @param string $role
	 *
	 * @return void
	 */
	public function theAdministratorHasEnabledTheRole(string $role): void {
		$roleId = GraphHelper::getPermissionsRoleIdByName($role);
		$defaultRoles = array_values(GraphHelper::DEFAULT_PERMISSIONS_ROLES);

		if (!\in_array($role, $defaultRoles)) {
			$defaultRoles[] = $roleId;
		}
		$envs = [
			"GRAPH_AVAILABLE_ROLES" => implode(',', $defaultRoles),
		];
		$response =  OcConfigHelper::reConfigureOc($envs);
		Assert::assertEquals(
			200,
			$response->getStatusCode(),
			"Failed to enable role $role"
		);
		$this->setEnabledPermissionsRoles($defaultRoles);
	}

	/**
	 * @Given the administrator has disabled the permissions role :role
	 *
	 * @param string $role
	 *
	 * @return void
	 */
	public function theAdministratorHasDisabledThePermissionsRole(string $role): void {
		$roleId = GraphHelper::getPermissionsRoleIdByName($role);
		$availableRoles = $this->getEnabledPermissionsRoles();

		if ($key = array_search($roleId, $availableRoles)) {
			unset($availableRoles[$key]);
		}
		$envs = [
			"GRAPH_AVAILABLE_ROLES" => implode(',', $availableRoles),
		];
		$response =  OcConfigHelper::reConfigureOc($envs);
		Assert::assertEquals(
			200,
			$response->getStatusCode(),
			"Failed to disable role $role"
		);
		$this->setEnabledPermissionsRoles($availableRoles);
	}

	/**
	 * @Given the config :configVariable has been set to path :path
	 *
	 * @param string $configVariable
	 * @param string $path
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function theConfigHasBeenSetPathTo(string $configVariable, string $path): void {
		if (\getenv('TEST_ROOT_PATH')) {
			$path = \getenv('TEST_ROOT_PATH') . "/" . $path;
		} else {
			$path = \realpath(\dirname(__FILE__) . "/../../" . $path);
		}

		$response =  OcConfigHelper::reConfigureOc(
			[
				$configVariable => $path
			]
		);
		Assert::assertEquals(
			200,
			$response->getStatusCode(),
			"Failed to set config $configVariable=$path"
		);
	}

	/**
	 * @Given the following configs have been set:
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function theConfigHasBeenSetToValue(TableNode $table): void {
		$envs = [];
		foreach ($table->getHash() as $row) {
			$envs[$row['config']] = $row['value'];
			if ($row['config'] === "POSTPROCESSING_DELAY") {
				OcConfigHelper::setPostProcessingDelay($row['value']);
			}
		}

		$response =  OcConfigHelper::reConfigureOc($envs);
		Assert::assertEquals(
			200,
			$response->getStatusCode(),
			"Failed to set config"
		);
	}

	/**
	 * @AfterScenario @env-config
	 *
	 * @return void
	 */
	public function rollbackOc(): void {
		OcConfigHelper::setPostProcessingDelay('0');
		$response = OcConfigHelper::rollbackOc();
		Assert::assertEquals(
			200,
			$response->getStatusCode(),
			"Failed to rollback OpenCloud server. Check if OpenCloud is started with ocwrapper."
		);
	}
}
