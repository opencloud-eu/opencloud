<?php declare(strict_types=1);
/**
 * @author    Artur Neumann <artur@jankaritech.com>
 * @copyright Copyright (c) 2020 Artur Neumann artur@jankaritech.com
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

namespace TestHelpers;

use Exception;
use GuzzleHttp\Exception\GuzzleException;

/**
 * Class StorageDriver
 *
 * @package TestHelpers
 */
abstract class StorageDriver {
	public const DECOMPOSED = "DECOMPOSED";
	public const EOS = "EOS";
	public const OWNCLOUD = "OWNCLOUD";
	public const DECOMPOSEDS3 = "DECOMPOSEDS3";
	public const POSIX = "POSIX";
}

/**
 * Class OcHelper
 *
 * Helper functions that are needed to run tests on OpenCloud server
 *
 * @package TestHelpers
 */
class OcHelper {
	public const STORAGE_DRIVERS = [
		StorageDriver::DECOMPOSED,
		StorageDriver::EOS,
		StorageDriver::OWNCLOUD,
		StorageDriver::DECOMPOSEDS3,
		StorageDriver::POSIX
	];

	/**
	 * @return string
	 */
	public static function getServerUrl(): string {
		if (\getenv('TEST_SERVER_URL')) {
			return \getenv('TEST_SERVER_URL');
		}
		return 'https://localhost:9200';
	}

	/**
	 * @return string
	 */
	public static function getFederatedServerUrl(): string {
		if (\getenv('TEST_SERVER_FED_URL')) {
			return \getenv('TEST_SERVER_FED_URL');
		}
		return 'https://localhost:10200';
	}

	/**
	 * @return string
	 */
	public static function getCollaborationServiceUrl(): string {
		if (\getenv("COLLABORATION_SERVICE_URL")) {
			return \getenv("COLLABORATION_SERVICE_URL");
		}
		return "http://localhost:9300";
	}

	/**
	 * @return bool
	 */
	public static function isTestingOnReva(): bool {
		return (\getenv("TEST_REVA") === "true");
	}

	/**
	 * @return bool
	 */
	public static function isUsingPreparedLdapUsers(): bool {
		return (\getenv("USE_PREPARED_LDAP_USERS") === "true");
	}

	/**
	 * @return bool|string false if no command given or the command as string
	 */
	public static function getDeleteUserDataCommand() {
		$cmd = \getenv("DELETE_USER_DATA_CMD");
		if ($cmd === false || \trim($cmd) === "") {
			return false;
		}
		return $cmd;
	}

	/**
	 * @return string
	 * @throws Exception
	 */
	public static function getStorageDriver(): string {
		$storageDriver = (\getenv("STORAGE_DRIVER"));
		if ($storageDriver === false) {
			return StorageDriver::DECOMPOSED;
		}
		$storageDriver = \strtoupper($storageDriver);
		if (!\in_array($storageDriver, self::STORAGE_DRIVERS)) {
			throw new Exception(
				"Invalid storage driver. " .
				"STORAGE_DRIVER must be '" . \join(", ", self::STORAGE_DRIVERS) . "'"
			);
		}
		return $storageDriver;
	}

	/**
	 * @param array $users
	 *
	 * @return void
	 * @throws Exception
	 */
	public static function deleteRevaUserData(?array $users = []): void {
		$deleteCmd = self::getDeleteUserDataCommand();

		if (self::getStorageDriver() === StorageDriver::POSIX) {
			\exec($deleteCmd);
			return;
		}

		foreach ($users as $user) {
			if (\is_array($user)) {
				$user = $user["actualUsername"];
			}
			if ($deleteCmd === false) {
				if (self::getStorageDriver() === StorageDriver::OWNCLOUD) {
					self::recurseRmdir(self::getOcRevaDataRoot() . $user);
				}
				continue;
			} elseif (self::getStorageDriver() === StorageDriver::EOS) {
				$deleteCmd = \str_replace(
					"%s",
					$user[0] . '/' . $user,
					$deleteCmd
				);
			} else {
				$deleteCmd = \sprintf($deleteCmd, $user);
			}
			\exec($deleteCmd);
		}
	}

	/**
	 * Helper for Recursive Copy of file/folder
	 * For more info check this out https://gist.github.com/gserrano/4c9648ec9eb293b9377b
	 *
	 * @param string|null $source
	 * @param string|null $destination
	 *
	 * @return void
	 */
	public static function recurseCopy(?string $source, ?string $destination): void {
		$dir = \opendir($source);
		@\mkdir($destination);
		while (($file = \readdir($dir)) !== false) {
			if (($file != '.') && ($file != '..')) {
				if (\is_dir($source . '/' . $file)) {
					self::recurseCopy($source . '/' . $file, $destination . '/' . $file);
				} else {
					\copy($source . '/' . $file, $destination . '/' . $file);
				}
			}
		}
		\closedir($dir);
	}

	/**
	 * @return int
	 */
	public static function getLdapPort(): int {
		$port = \getenv("REVA_LDAP_PORT");
		return $port ? (int)$port : 636;
	}

	/**
	 * @return bool
	 */
	public static function useSsl(): bool {
		$useSsl = \getenv("REVA_LDAP_USESSL");
		if ($useSsl === false) {
			return (self::getLdapPort() === 636);
		} else {
			return $useSsl === "true";
		}
	}

	/**
	 * @return string
	 */
	public static function getBaseDN(): string {
		$dn = \getenv("REVA_LDAP_BASE_DN");
		return $dn ?: "dc=example,dc=org";
	}

	/**
	 * @return string
	 */
	public static function getGroupsOU(): string {
		$ou = \getenv("REVA_LDAP_GROUPS_OU");
		return $ou ?: "TestGroups";
	}

	/**
	 * @return string
	 */
	public static function getUsersOU(): string {
		$ou = \getenv("REVA_LDAP_USERS_OU");
		return $ou ?: "TestUsers";
	}

	/**
	 * @return string
	 */
	public static function getGroupSchema(): string {
		$schema = \getenv("REVA_LDAP_GROUP_SCHEMA");
		return $schema ?: "rfc2307";
	}
	/**
	 * @return string
	 */
	public static function getHostname(): string {
		$hostname = \getenv("REVA_LDAP_HOSTNAME");
		return $hostname ?: "localhost";
	}

	/**
	 * @return string
	 */
	public static function getBindDN(): string {
		$dn = \getenv("REVA_LDAP_BIND_DN");
		return $dn ?: "cn=admin,dc=example,dc=org";
	}

	/**
	 * @return string
	 */
	public static function getBindPassword(): string {
		$pw = \getenv("REVA_LDAP_BIND_PASSWORD");
		return $pw ?: "";
	}

	/**
	 * @return string
	 */
	private static function getOcRevaDataRoot(): string {
		$root = \getenv("OC_REVA_DATA_ROOT");
		if ($root === false || $root === "") {
			$root = "/var/tmp/opencloud/opencloud/";
		}
		if (!\file_exists($root)) {
			echo "WARNING: reva data root folder ($root) does not exist\n";
		}
		return $root;
	}

	/**
	 * @param string|null $dir
	 *
	 * @return bool
	 */
	private static function recurseRmdir(?string $dir): bool {
		if (\file_exists($dir) === true) {
			$files = \array_diff(\scandir($dir), ['.', '..']);
			foreach ($files as $file) {
				if (\is_dir("$dir/$file")) {
					self::recurseRmdir("$dir/$file");
				} else {
					\unlink("$dir/$file");
				}
			}
			return \rmdir($dir);
		}
		return true;
	}

	/**
	 * On Eos storage backend when the user data is cleared after test run
	 * Running another test immediately fails. So Send this request to create user home directory
	 *
	 * @param string|null $baseUrl
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $xRequestId
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public static function createEOSStorageHome(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $xRequestId = ''
	): void {
		HttpRequestHelper::get(
			$baseUrl . "/ocs/v2.php/apps/notifications/api/v1/notifications",
			$xRequestId,
			$user,
			$password
		);
	}
}
