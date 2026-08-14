<?php declare(strict_types=1);
/**
 * @author Sajan Gurung <sajan@jankaritech.com>
 * @copyright Copyright (c) 2024 Sajan Gurung sajan@jankaritech.com
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

use TestHelpers\HttpRequestHelper;
use GuzzleHttp\Exception\GuzzleException;
use Psr\Http\Message\ResponseInterface;
use PHPUnit\Framework\Assert;

/**
 * A helper class for OpenCloud settings
 */
class SettingsHelper {
	private static string $settingsEndpoint = '/api/v0/settings/';
	private const BUNDLE_ID = "2a506de7-99bd-4f0d-994e-c38e72c28fd9";

	/**
	 * @param string $baseUrl
	 * @param string $path
	 *
	 * @return string
	 */
	public static function buildFullUrl(string $baseUrl, string $path): string {
		return $baseUrl . self::$settingsEndpoint . $path;
	}

	/**
	 * @return string
	 */
	public static function getBundleId(): string {
		return self::BUNDLE_ID;
	}

	/**
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $xRequestId
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getBundlesList(
		string $baseUrl,
		string $user,
		string $password,
		string $xRequestId,
		array $headers = []
	): ResponseInterface {
		$fullUrl = self::buildFullUrl($baseUrl, "bundles-list");
		return HttpRequestHelper::post(
			$fullUrl,
			$xRequestId,
			$user,
			$password,
			$headers,
			"{}"
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $bundleName
	 * @param string $xRequestId
	 *
	 * @return array
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getBundleByName(
		string $baseUrl,
		string $user,
		string $password,
		string $bundleName,
		string $xRequestId
	): array {
		$response = self::getBundlesList($baseUrl, $user, $password, $xRequestId);
		Assert::assertEquals(201, $response->getStatusCode(), "Failed to get bundles list");

		$bundlesList = HttpRequestHelper::getJsonDecodedResponseBodyContent($response);
		foreach ($bundlesList->bundles as $value) {
			if ($value->displayName === $bundleName) {
				return json_decode(json_encode($value), true);
			}
		}
		return [];
	}

	/**
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $xRequestId
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getRolesList(
		string $baseUrl,
		string $user,
		string $password,
		string $xRequestId,
		array $headers = []
	): ResponseInterface {
		$fullUrl = self::buildFullUrl($baseUrl, "roles-list");
		return HttpRequestHelper::post(
			$fullUrl,
			$xRequestId,
			$user,
			$password,
			$headers,
			"{}"
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $assigneeId
	 * @param string $roleId
	 * @param string $xRequestId
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function assignRoleToUser(
		string $baseUrl,
		string $user,
		string $password,
		string $assigneeId,
		string $roleId,
		string $xRequestId,
		array $headers = []
	): ResponseInterface {
		$fullUrl = self::buildFullUrl($baseUrl, "assignments-add");
		$body = json_encode(["account_uuid" => $assigneeId, "role_id" => $roleId], JSON_THROW_ON_ERROR);
		return HttpRequestHelper::post(
			$fullUrl,
			$xRequestId,
			$user,
			$password,
			$headers,
			$body
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $userId
	 * @param string $xRequestId
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getAssignmentsList(
		string $baseUrl,
		string $user,
		string $password,
		string $userId,
		string $xRequestId,
		array $headers = []
	): ResponseInterface {
		$fullUrl = self::buildFullUrl($baseUrl, "assignments-list");
		$body = json_encode(["account_uuid" => $userId], JSON_THROW_ON_ERROR);
		return HttpRequestHelper::post(
			$fullUrl,
			$xRequestId,
			$user,
			$password,
			$headers,
			$body
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $xRequestId
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getValuesList(
		string $baseUrl,
		string $user,
		string $password,
		string $xRequestId,
		array $headers = []
	): ResponseInterface {
		$fullUrl = self::buildFullUrl($baseUrl, "values-list");
		$body = json_encode(["account_uuid" => "me"], JSON_THROW_ON_ERROR);
		return HttpRequestHelper::post(
			$fullUrl,
			$xRequestId,
			$user,
			$password,
			$headers,
			$body
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $xRequestId
	 *
	 * @return bool
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getAutoAcceptSharesSettingValue(
		string $baseUrl,
		string $user,
		string $password,
		string $xRequestId
	): bool {
		$response = self::getValuesList($baseUrl, $user, $password, $xRequestId);
		Assert::assertEquals(201, $response->getStatusCode(), "Failed to get values list");

		$valuesList = HttpRequestHelper::getJsonDecodedResponseBodyContent($response);

		if (empty($valuesList)) {
			return true;
		}

		foreach ($valuesList->values as $value) {
			if ($value->identifier->setting === "auto-accept-shares") {
				return $value->value->boolValue;
			}
		}

		return true;
	}

	/**
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $xRequestId
	 *
	 * @return string
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getLanguageSettingValue(
		string $baseUrl,
		string $user,
		string $password,
		string $xRequestId
	): string {
		$response = self::getValuesList($baseUrl, $user, $password, $xRequestId);
		Assert::assertEquals(201, $response->getStatusCode(), "Failed to get values list");

		$valuesList = HttpRequestHelper::getJsonDecodedResponseBodyContent($response);

		// if no language is set, the request body is empty return English as the default language
		if (empty($valuesList)) {
			return "en";
		}
		foreach ($valuesList->values as $value) {
			if ($value->identifier->setting === "language") {
				return $value->value->listValue->values[0]->stringValue;
			}
		}
		// if a language setting was still not found, return English
		return "en";
	}

	/**
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $body
	 * @param string $xRequestId
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function updateSettings(
		string $baseUrl,
		string $user,
		string $password,
		string $body,
		string $xRequestId,
		array $headers = []
	): ResponseInterface {
		$fullUrl = self::buildFullUrl($baseUrl, "values-save");
		return HttpRequestHelper::post(
			$fullUrl,
			$xRequestId,
			$user,
			$password,
			$headers,
			$body
		);
	}
}
