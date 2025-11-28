<?php declare(strict_types=1);
/**
 * @author Kiran Parajuli <kiran@jankaritech.com>
 * @copyright Copyright (c) 2022 Kiran Parajuli kiran@jankaritech.com
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

use GuzzleHttp\Exception\GuzzleException;
use Psr\Http\Message\RequestInterface;
use Psr\Http\Message\ResponseInterface;

/**
 * A helper class for managing Graph API requests
 */
class GraphHelper {
	public const DEFAULT_PERMISSIONS_ROLES = [
		'Viewer' => 'b1e2218d-eef8-4d4c-b82d-0f1a1b48f3b5',
		'Editor' => 'fb6c3e19-e378-47e5-b277-9732f9de6e21',
		'File Editor' => '2d00ce52-1fc2-4dbc-8b95-a73b73395f5a',
		'Uploader' => '1c996275-f1c9-4e71-abdf-a42f6495e960',
		'Space Viewer' => 'a8d5fe5e-96e3-418d-825b-534dbdf22b99',
		'Space Editor' => '58c63c02-1d89-4572-916a-870abc5a1b7d',
		'Manager' => '312c0871-5ef7-4b3a-85b6-0e4074c64049',
	];

	public const ADDITIONAL_PERMISSIONS_ROLES = [
		'Secure Viewer' => 'aa97fe03-7980-45ac-9e50-b325749fd7e6',
		'Space Editor Without Versions' => '3284f2d5-0070-4ad8-ac40-c247f7c1fb27',
		'Denied' => '63e64e19-8d43-42ec-a738-2b6af2610efa',
	];

	public const SHARES_SPACE_ID = 'a0ca6a90-a365-4782-871e-d44447bbc668$a0ca6a90-a365-4782-871e-d44447bbc668';

	/**
	 * @return string[]
	 */
	private static function getRequestHeaders(): array {
		return [
			'Content-Type' => 'application/json',
		];
	}

	/**
	 *
	 * @return string
	 */
	public static function getUUIDv4Regex(): string {
		return '[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-4[0-9A-Fa-f]{3}-[89ABab][0-9A-Fa-f]{3}-[0-9A-Fa-f]{12}';
	}

	/**
	 * @param string $id
	 *
	 * @return bool
	 */
	public static function isUUIDv4(string $id): bool {
		$regex = "/^" . self::getUUIDv4Regex() . "$/i";
		return (bool)preg_match($regex, $id);
	}

	/**
	 * @param string $spaceId
	 *
	 * @return bool
	 */
	public static function isSpaceId(string $spaceId): bool {
		$regex = "/^" . self::getUUIDv4Regex() . '\\$' . self::getUUIDv4Regex() . "$/i";
		return (bool)preg_match($regex, $spaceId);
	}

	/**
	 * @return string
	 */
	public static function getSpaceIdRegex(): string {
		return self::getUUIDv4Regex() . '\\\$' . self::getUUIDv4Regex();
	}

	/**
	 * @return string
	 */
	public static function getPermissionsIdRegex(): string {
		return self::getUUIDv4Regex() . ':' . self::getUUIDv4Regex() . ':' . self::getUUIDv4Regex();
	}

	/**
	 * @return string
	 */
	public static function getFileIdRegex(): string {
		return self::getUUIDv4Regex() . '\\\$' . self::getUUIDv4Regex() . '!' . self::getUUIDv4Regex();
	}

	/**
	 * @return string
	 */
	public static function getShareIdRegex(): string {
		return self::getUUIDv4Regex() . '\\\$' . self::getUUIDv4Regex() . '!'
		. self::getUUIDv4Regex() . ':' . self::getUUIDv4Regex() . ':' . self::getUUIDv4Regex();
	}

	/**
	 * @return string
	 */
	public static function getEtagRegex(): string {
		return "^\\\"[a-f0-9:.]{1,32}\\\"$";
	}

	/**
	 * Federated users have a string of {userid}@{provider} as their id
	 *
	 * @return string
	 */
	public static function getFederatedUserRegex(): string {
		return '^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}@https?://.+$';
	}

	/**
	 * @return string
	 */
	public static function getFederatedFileIdRegex(): string {
		return self::getSpaceIdRegex() . '!(?=(.{4})*$)[A-Za-z0-9+/]*={0,2}';
	}

	/**
	 * @param string $pattern
	 *
	 * @return string regex pattern
	 */
	public static function jsonSchemaRegexToPureRegex(string $pattern): string {
		$pattern = \str_replace("\\\\", "\\", $pattern);
		$pattern = \str_replace("/", "\/", $pattern);
		$pattern = \preg_replace('/^\^/', '', $pattern);
		$pattern = \preg_replace('/\$$/', '', $pattern);
		return "/^$pattern$/";
	}

	/**
	 * Key name can consist of @@@
	 * This function separate such key and return its actual value from actual drive response which can be used for assertion
	 *
	 * @param string $keyName
	 * @param array $actualDriveInformation
	 *
	 * @return string
	 */
	public static function separateAndGetValueForKey(string $keyName, array $actualDriveInformation): string {
		// break the segment with @@@ to find the actual value from the actual drive information
		$separatedKey = explode("@@@", $keyName);
		// this stores the actual value of each key from drive information response used for assertion
		$actualKeyValue = $actualDriveInformation;

		foreach ($separatedKey as $key) {
			$actualKeyValue = $actualKeyValue[$key];
		}

		return $actualKeyValue;
	}

	/**
	 * @param string $baseUrl
	 * @param string $path
	 *
	 * @return string
	 */
	public static function getFullUrl(string $baseUrl, string $path): string {
		$fullUrl = $baseUrl;
		if (\substr($fullUrl, -1) !== '/') {
			$fullUrl .= '/';
		}
		$fullUrl .= 'graph/v1.0/' . $path;
		return $fullUrl;
	}

	/**
	 * @param string $baseUrl
	 * @param string $path
	 *
	 * @return string
	 */
	public static function getBetaFullUrl(string $baseUrl, string $path): string {
		$baseUrl = rtrim($baseUrl, "/");
		return $baseUrl . '/graph/v1beta1/' . $path;
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $userName
	 * @param string $password
	 * @param string|null $email
	 * @param string|null $displayName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function createUser(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $userName,
		string $password,
		?string $email = null,
		?string $displayName = null
	): ResponseInterface {
		$payload = self::prepareCreateUserPayload(
			$userName,
			$password,
			$email,
			$displayName
		);

		$url = self::getFullUrl($baseUrl, 'users');
		return HttpRequestHelper::post(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
			self::getRequestHeaders(),
			$payload
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $userId
	 * @param string $method
	 * @param string|null $userName
	 * @param string|null $password
	 * @param string|null $email
	 * @param string|null $displayName
	 * @param bool|true $accountEnabled
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function editUser(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $userId,
		?string $method = "PATCH",
		?string $userName = null,
		?string $password = null,
		?string $email = null,
		?string $displayName = null,
		?bool $accountEnabled = true
	): ResponseInterface {
		$payload = self::preparePatchUserPayload(
			$userName,
			$password,
			$email,
			$displayName,
			$accountEnabled
		);
		$url = self::getFullUrl($baseUrl, 'users/' . $userId);
		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			$method,
			$adminUser,
			$adminPassword,
			self::getRequestHeaders(),
			$payload
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $userName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getUser(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $userName
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users/' . $userName);
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $searchTerm
	 *
	 * @return ResponseInterface
	 */
	public static function searchUser(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $searchTerm
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, "users?\$search=$searchTerm");
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $searchTerm
	 *
	 * @return ResponseInterface
	 */
	public static function searchFederatedUser(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $searchTerm
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, "users?\$filter=userType eq 'Federated'&\$search=$searchTerm");
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $userPassword
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getOwnInformationAndGroupMemberships(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $userPassword
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'me/?%24expand=memberOf');
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$userPassword,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $userName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function deleteUser(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $userName
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users/' . $userName);
		return HttpRequestHelper::delete(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $userId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function deleteUserByUserId(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $userId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users/' . $userId);

		return HttpRequestHelper::delete(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $byUser
	 * @param string $userPassword
	 * @param string|null $userIdOrName User ID or username to get drive information for
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getUserWithDriveInformation(
		string $baseUrl,
		string $xRequestId,
		string $byUser,
		string $userPassword,
		?string $userIdOrName = null
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users/' . $userIdOrName . '?%24select=&%24expand=drive');
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$byUser,
			$userPassword,
		);
	}

	/***
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $byUser
	 * @param string $userPassword
	 * @param string $userId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getPersonalDriveInformationByUserId(
		string $baseUrl,
		string $xRequestId,
		string $byUser,
		string $userPassword,
		string $userId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users/' . $userId . '/drive');
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$byUser,
			$userPassword
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $byUser
	 * @param string $userPassword
	 * @param string|null $user
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getUserWithGroupInformation(
		string $baseUrl,
		string $xRequestId,
		string $byUser,
		string $userPassword,
		?string $user = null
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users/' . $user . '?%24expand=memberOf');
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$byUser,
			$userPassword,
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $groupName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function createGroup(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $groupName
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'groups');
		$payload['displayName'] = $groupName;
		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			"POST",
			$adminUser,
			$adminPassword,
			self::getRequestHeaders(),
			\json_encode($payload)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $groupId
	 * @param string $displayName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function updateGroup(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $groupId,
		string $displayName
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'groups/' . $groupId);
		$payload['displayName'] = $displayName;
		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			"PATCH",
			$adminUser,
			$adminPassword,
			self::getRequestHeaders(),
			\json_encode($payload)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getUsers(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users');
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
			self::getRequestHeaders(),
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getGroups(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'groups');
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
			self::getRequestHeaders(),
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $groupName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getGroup(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $groupName
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'groups/' . $groupName);
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $searchTerm
	 *
	 * @return ResponseInterface
	 */
	public static function searchGroup(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $searchTerm
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, "groups?\$search=$searchTerm");
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $groupId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function deleteGroup(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $groupId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'groups/' . $groupId);
		return HttpRequestHelper::delete(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
		);
	}

	/**
	 * add multiple users to a group at once
	 *
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $groupId
	 * @param array $userIds
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function addUsersToGroup(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $groupId,
		array $userIds
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'groups/' . $groupId);
		$payload = [ "members@odata.bind" => [] ];
		foreach ($userIds as $userId) {
			$payload["members@odata.bind"][] = self::getFullUrl($baseUrl, 'users/' . $userId);
		}
		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			'PATCH',
			$adminUser,
			$adminPassword,
			self::getRequestHeaders(),
			\json_encode($payload)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $userId
	 * @param string $groupId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function addUserToGroup(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $userId,
		string $groupId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'groups/' . $groupId . '/members/$ref');
		$body = [
			"@odata.id" => self::getFullUrl($baseUrl, 'users/' . $userId)
		];
		return HttpRequestHelper::post(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
			self::getRequestHeaders(),
			\json_encode($body)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $userId
	 * @param string $groupId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function removeUserFromGroup(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $userId,
		string $groupId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'groups/' . $groupId . '/members/' . $userId . '/$ref');
		return HttpRequestHelper::delete(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string $groupId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getMembersList(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		string $groupId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'groups/' . $groupId . '/members');
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword
		);
	}

	/**
	 * returns single group information along with its member information when groupId is provided
	 * else return all group information along with its member information
	 *
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 * @param string|null $groupId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getSingleOrAllGroupsAlongWithMembers(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
		?string $groupId = null
	): ResponseInterface {
		// we can expand to get list of members for a single group with groupId and also expand to get all groups with all its members
		$endPath = ($groupId) ? '/' . $groupId . '?$expand=members' : '?$expand=members';
		$url = self::getFullUrl($baseUrl, 'groups' . $endPath);
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword
		);
	}

	/**
	 * returns json encoded payload for user creating request
	 *
	 * @param string|null $userName
	 * @param string|null $password
	 * @param string|null $email
	 * @param string|null $displayName
	 *
	 * @return string
	 */
	public static function prepareCreateUserPayload(
		string $userName,
		string $password,
		?string $email,
		?string $displayName
	): string {
		$payload['onPremisesSamAccountName'] = $userName;
		$payload['passwordProfile'] = ['password' => $password];
		$payload['displayName'] = $displayName ?? $userName;
		if (empty($email)) {
			$payload['mail'] = $userName . '@example.com';
		} else {
			$payload['mail'] = $email;
		}
		$payload['accountEnabled'] = true;
		return \json_encode($payload);
	}

	/**
	 * returns encoded json payload for user patching requests
	 *
	 * @param string|null $userName
	 * @param string|null $password
	 * @param string|null $email
	 * @param string|null $displayName
	 * @param bool|true $accountEnabled
	 *
	 * @return string
	 */
	public static function preparePatchUserPayload(
		?string $userName,
		?string $password,
		?string $email,
		?string $displayName,
		?bool $accountEnabled
	): string {
		$payload = [];
		if ($userName !== null) {
			// comment on after fixing #5755 because now it crashes server
			// if (empty($userName)) {
			//   $payload['onPremisesSamAccountName'] = ' ';
			// } else $payload['onPremisesSamAccountName'] = $userName;
			$payload['onPremisesSamAccountName'] = $userName;
		}
		if ($password !== null) {
			$payload['passwordProfile'] = ['password' => $password];
		}
		if ($displayName !== null) {
			$payload['displayName'] = $displayName;
		}
		if ($email !== null) {
			$payload['mail'] = $email;
		}
		$payload['accountEnabled'] = $accountEnabled;
		return \json_encode($payload);
	}

	/**
	 * Send Graph Create Space Request
	 *
	 * @param string $baseUrl
	 * @param  string $user
	 * @param  string $password
	 * @param  string $body
	 * @param  string $xRequestId
	 * @param  array  $headers
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function createSpace(
		string $baseUrl,
		string $user,
		string $password,
		string $body,
		string $xRequestId = '',
		array $headers = []
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'drives');

		return HttpRequestHelper::post($url, $xRequestId, $user, $password, $headers, $body);
	}

	/**
	 * Send Graph Update Space Request
	 *
	 * @param  string $baseUrl
	 * @param  string $user
	 * @param  string $password
	 * @param  mixed $body
	 * @param  string $spaceId
	 * @param  string $xRequestId
	 * @param  array  $headers
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function updateSpace(
		string $baseUrl,
		string $user,
		string $password,
		$body,
		string $spaceId,
		string $xRequestId = '',
		array $headers = []
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'drives/' . $spaceId);

		return HttpRequestHelper::sendRequest($url, $xRequestId, 'PATCH', $user, $password, $headers, $body);
	}

	/**
	 * Send Graph List My Spaces Request
	 *
	 * @param  string $baseUrl
	 * @param  string $user
	 * @param  string $password
	 * @param  string $urlArguments
	 * @param  string $xRequestId
	 * @param  array  $body
	 * @param  array  $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public static function getMySpaces(
		string $baseUrl,
		string $user,
		string $password,
		string $urlArguments = '',
		string $xRequestId = '',
		array  $body = [],
		array  $headers = []
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'me/drives' . $urlArguments);

		return HttpRequestHelper::get($url, $xRequestId, $user, $password, $headers, $body);
	}

	/**
	 * Send Graph List All Spaces Request
	 *
	 * @param  string $baseUrl
	 * @param  string $user
	 * @param  string $password
	 * @param  string $urlArguments
	 * @param  string $xRequestId
	 * @param  array  $body
	 * @param  array  $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public static function getAllSpaces(
		string $baseUrl,
		string $user,
		string $password,
		string $urlArguments = '',
		string $xRequestId = '',
		array  $body = [],
		array  $headers = []
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'drives/' . $urlArguments);

		return HttpRequestHelper::get($url, $xRequestId, $user, $password, $headers, $body);
	}

	/**
	 * Send Graph List Single Space Request
	 *
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param string $urlArguments
	 * @param string $xRequestId
	 * @param array $body
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public static function getSingleSpace(
		string $baseUrl,
		string $user,
		string $password,
		string $spaceId,
		string $urlArguments = '',
		string $xRequestId = '',
		array  $body = [],
		array  $headers = []
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'drives/' . $spaceId . "/" . $urlArguments);

		return HttpRequestHelper::get($url, $xRequestId, $user, $password, $headers, $body);
	}

	/**
	 * send disable space request
	 *
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param string $xRequestId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function disableSpace(
		string $baseUrl,
		string $user,
		string $password,
		string $spaceId,
		string $xRequestId = ''
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'drives/' . $spaceId);

		return HttpRequestHelper::delete(
			$url,
			$xRequestId,
			$user,
			$password
		);
	}

	/**
	 * send delete space request
	 *
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param string $xRequestId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function deleteSpace(
		string $baseUrl,
		string $user,
		string $password,
		string $spaceId,
		string $xRequestId = ''
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'drives/' . $spaceId);
		$header = ["Purge" => "T"];

		return HttpRequestHelper::delete(
			$url,
			$xRequestId,
			$user,
			$password,
			$header
		);
	}

	/**
	 * Send restore Space Request
	 *
	 * @param  string $baseUrl
	 * @param  string $user
	 * @param  string $password
	 * @param  string $spaceId
	 * @param string $xRequestId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function restoreSpace(
		string $baseUrl,
		string $user,
		string $password,
		string $spaceId,
		string $xRequestId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'drives/' . $spaceId);
		$header = ["restore" => true];
		$body = '{}';

		return HttpRequestHelper::sendRequest($url, $xRequestId, 'PATCH', $user, $password, $header, $body);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $currentPassword
	 * @param string $newPassword
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function changeOwnPassword(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $currentPassword,
		string $newPassword
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'me/changePassword');
		$payload['currentPassword'] = $currentPassword;
		$payload['newPassword'] = $newPassword;

		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			"POST",
			$user,
			$password,
			self::getRequestHeaders(),
			\json_encode($payload)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $xRequestId
	 * @param array $body
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getTags(
		string $baseUrl,
		string $user,
		string $password,
		string $xRequestId = '',
		array  $body = [],
		array  $headers = []
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'extensions/org.libregraph/tags');

		return HttpRequestHelper::get($url, $xRequestId, $user, $password, $headers, $body);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $resourceId
	 * @param array $tagName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function createTags(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $resourceId,
		array $tagName
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'extensions/org.libregraph/tags');
		$payload['resourceId'] = $resourceId;
		$payload['tags'] = $tagName;

		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			"PUT",
			$user,
			$password,
			self::getRequestHeaders(),
			\json_encode($payload)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $resourceId
	 * @param array $tagName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function deleteTags(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $resourceId,
		array $tagName
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'extensions/org.libregraph/tags');
		$payload['resourceId'] = $resourceId;
		$payload['tags'] = $tagName;

		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			"DELETE",
			$user,
			$password,
			self::getRequestHeaders(),
			\json_encode($payload)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getApplications(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'applications');
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $groupId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getUsersWithFilterMemberOf(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $groupId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users' . '?$filter=memberOf/any(m:m/id ' . "eq '$groupId')");
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param array $groupIdArray
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getUsersOfTwoGroups(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		array $groupIdArray
	): ResponseInterface {
		$url = self::getFullUrl(
			$baseUrl,
			'users' . '?$filter=memberOf/any(m:m/id ' . "eq '$groupIdArray[0]') "
			. "and memberOf/any(m:m/id eq '$groupIdArray[1]')"
		);
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $firstGroup
	 * @param string $secondGroup
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getUsersFromOneOrOtherGroup(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $firstGroup,
		string $secondGroup
	): ResponseInterface {
		$url = self::getFullUrl(
			$baseUrl,
			'users' . '?$filter=memberOf/any(m:m/id '
			. "eq '$firstGroup') " . "or memberOf/any(m:m/id eq '$secondGroup')"
		);
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $roleId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getUsersWithFilterRoleAssignment(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $roleId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users' . '?$filter=appRoleAssignments/any(m:m/appRoleId ' . "eq '$roleId')");
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $roleId
	 * @param string $groupId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getUsersWithFilterRolesAssignmentAndMemberOf(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $roleId,
		string $groupId
	): ResponseInterface {
		$url = self::getFullUrl(
			$baseUrl,
			'users' . '?$filter=appRoleAssignments/any(m:m/appRoleId '
			. "eq '$roleId') " . "and memberOf/any(m:m/id eq '$groupId')"
		);
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $appRoleId
	 * @param string $applicationId
	 * @param string $userId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function assignRole(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $appRoleId,
		string $applicationId,
		string $userId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users/' . $userId . '/appRoleAssignments');
		$payload['principalId'] = $userId;
		$payload['appRoleId'] = $appRoleId;
		$payload['resourceId'] = $applicationId;
		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			"POST",
			$user,
			$password,
			self::getRequestHeaders(),
			\json_encode($payload)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $userId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getAssignedRole(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $userId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users/' . $userId . '/appRoleAssignments');
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $userId
	 * @param string $path
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function generateGDPRReport(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $userId,
		string $path
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users/' . $userId . '/exportPersonalData');
		// this payload is the storage location of the report generated
		$payload['storageLocation'] = $path;
		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			"POST",
			$user,
			$password,
			self::getRequestHeaders(),
			\json_encode($payload)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $appRoleAssignmentId
	 * @param string $userId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function unassignRole(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $appRoleAssignmentId,
		string $userId
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, 'users/' . $userId . '/appRoleAssignments/' . $appRoleAssignmentId);
		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			"DELETE",
			$user,
			$password,
			self::getRequestHeaders(),
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $path
	 *
	 * @return string
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getShareMountId(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $path
	): string {
		$response = self::getMySpaces(
			$baseUrl,
			$user,
			$password,
			'',
			$xRequestId
		);
		$drives = json_decode($response->getBody()->getContents(), true, 512, JSON_THROW_ON_ERROR);

		// the response returns the shared resource in driveAlias all in lowercase,
		// For example: if we get the property of a shared resource "FOLDER" then the response contains "driveAlias": "mountpoint/folder"
		// In case of two shares with same name, the response for the second shared resource will contain, "driveAlias": "mountpoint/folder-(2)"
		$path = strtolower($path);
		foreach ($drives["value"] as $value) {
			if ($value["driveAlias"] === "mountpoint/" . $path) {
				return $value["id"];
			}
		}
		throw new \Exception(
			__METHOD__
			. " Cannot find share mountpoint id of '$path' for user '$user'"
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $language
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function switchSystemLanguage(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $language
	): ResponseInterface {
		$fullUrl = self::getFullUrl($baseUrl, 'me');
		$payload['preferredLanguage'] = $language;
		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$xRequestId,
			'PATCH',
			$user,
			$password,
			null,
			\json_encode($payload)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param string $itemId
	 * @param string|null $query
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getPermissionsList(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		string $itemId,
		?string $query = null
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/items/$itemId/permissions");
		if (!empty($query)) {
			$url .= "?$query";
		}

		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * Get the role id by name
	 *
	 * @param string $permissionsRole
	 *
	 * @return string
	 *
	 * @throws \Exception
	 */
	public static function getPermissionsRoleIdByName(string $permissionsRole): string {
		$allPermissionsRoles = array_merge(self::DEFAULT_PERMISSIONS_ROLES, self::ADDITIONAL_PERMISSIONS_ROLES);
		if (\array_key_exists($permissionsRole, $allPermissionsRoles)) {
			return $allPermissionsRoles[$permissionsRole];
		}

		throw new \Exception("Role '$permissionsRole' not found");
	}

	/**
	 * Get the role name by role id
	 *
	 * @param string $permissionsRoleId
	 *
	 * @return string
	 *
	 * @throws \Exception
	 */
	public static function getPermissionNameByPermissionRoleId(string $permissionsRoleId): string {
		$allPermissionsRoles = array_merge(self::DEFAULT_PERMISSIONS_ROLES, self::ADDITIONAL_PERMISSIONS_ROLES);
		$roleName = array_search($permissionsRoleId, $allPermissionsRoles);
		if ($roleName) {
			return $roleName;
		}
		throw new \Exception("Permission role id: '$permissionsRoleId' not found");
	}

	/**
	 * @param array $shareeIds
	 * @param array $shareTypes
	 * @param string|null $permissionsRole
	 * @param string|null $permissionsAction
	 * @param string|null $expirationDateTime
	 *
	 * @return array
	 * @throws \Exception
	 */
	public static function createShareInviteBody(
		array $shareeIds,
		array $shareTypes,
		?string $permissionsRole,
		?string $permissionsAction,
		?string $expirationDateTime
	): array {
		$body = [];

		foreach ($shareeIds as $index => $shareeId) {
			$shareType = $shareTypes[$index];
			$body['recipients'][] = [
				"@libre.graph.recipient.type" => $shareType,
				"objectId" => $shareeId
			];
		}

		if ($permissionsRole !== null) {
			$roleId = self::getPermissionsRoleIdByName($permissionsRole);
			$body['roles'] = [$roleId];
		}

		if ($permissionsAction !== null) {
			$body['@libre.graph.permissions.actions'] = ['libre.graph/driveItem/' . $permissionsAction];
		}

		if ($expirationDateTime !== null) {
			$body['expirationDateTime'] = $expirationDateTime;
		}
		return $body;
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param string $itemId
	 * @param array $shareeIds
	 * @param array $shareTypes
	 * @param string|null $permissionsRole
	 * @param string|null $permissionsAction
	 * @param string|null $expirationDateTime
	 *
	 * @return ResponseInterface
	 * @throws \JsonException
	 * @throws \Exception
	 */
	public static function sendSharingInvitation(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		string $itemId,
		array $shareeIds,
		array $shareTypes,
		?string $permissionsRole,
		?string $permissionsAction,
		?string $expirationDateTime
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/items/$itemId/invite");
		$body = self::createShareInviteBody(
			$shareeIds,
			$shareTypes,
			$permissionsRole,
			$permissionsAction,
			$expirationDateTime
		);
		return HttpRequestHelper::post(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders(),
			\json_encode($body)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param string $itemId
	 * @param mixed $body
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function createLinkShare(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		string $itemId,
		$body
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/items/$itemId/createLink");
		return HttpRequestHelper::post(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders(),
			$body
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param string $itemId
	 * @param mixed $body
	 * @param string $permissionsId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function updateShare(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		string $itemId,
		$body,
		string $permissionsId
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/items/$itemId/permissions/$permissionsId");
		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			'PATCH',
			$user,
			$password,
			self::getRequestHeaders(),
			$body
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param string $itemId
	 * @param mixed $body
	 * @param string $permissionsId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function setLinkSharePassword(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		string $itemId,
		$body,
		string $permissionsId
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/items/$itemId/permissions/$permissionsId/setPassword");
		return HttpRequestHelper::post(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders(),
			$body
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param string $itemId
	 * @param string $permissionId
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public static function removeAccessToSpaceItem(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		string $itemId,
		string $permissionId
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/items/$itemId/permissions/$permissionId");
		return HttpRequestHelper::delete(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param string $permissionId
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public static function removeAccessToSpace(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		string $permissionId
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/root/permissions/$permissionId");
		return HttpRequestHelper::delete(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getSharesSharedWithMe(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "me/drive/sharedWithMe?\$expand=thumbnails");
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getSharesSharedByMe(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "me/drive/sharedByMe?\$expand=thumbnails");
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $itemId
	 * @param string $shareSpaceId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function disableShareSync(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $itemId,
		string $shareSpaceId
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$shareSpaceId/items/$itemId");
		return HttpRequestHelper::delete(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $itemId
	 * @param string $shareSpaceId
	 * @param array $body
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function hideOrUnhideShare(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $itemId,
		string $shareSpaceId,
		array $body
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$shareSpaceId/items/$itemId");
		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			'PATCH',
			$user,
			$password,
			self::getRequestHeaders(),
			\json_encode($body)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $itemId
	 * @param string $shareSpaceId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function enableShareSync(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $itemId,
		string $shareSpaceId
	): ResponseInterface {
		$body = [
			"remoteItem" => [
				"id" => $itemId
			]
		];
		$url = self::getBetaFullUrl($baseUrl, "drives/$shareSpaceId/root/children");
		return HttpRequestHelper::post(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders(),
			\json_encode($body)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param string|null $query
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getDrivePermissionsList(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		?string $query = null
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/root/permissions");
		if ($query !== null) {
			$url .= "?$query";
		}

		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param array $shareeIds
	 * @param array $shareTypes
	 * @param string|null $permissionsRole
	 * @param string|null $permissionsAction
	 * @param string|null $expirationDateTime
	 *
	 * @return ResponseInterface
	 * @throws \Exception|GuzzleException
	 */
	public static function sendSharingInvitationForDrive(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		array $shareeIds,
		array $shareTypes,
		?string $permissionsRole,
		?string $permissionsAction,
		?string $expirationDateTime
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/root/invite");
		$body = self::createShareInviteBody(
			$shareeIds,
			$shareTypes,
			$permissionsRole,
			$permissionsAction,
			$expirationDateTime
		);

		return HttpRequestHelper::post(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders(),
			\json_encode($body)
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param mixed $body
	 * @param string $permissionsId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function updateDriveShare(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		$body,
		string $permissionsId
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/root/permissions/$permissionsId");

		return HttpRequestHelper::sendRequest(
			$url,
			$xRequestId,
			'PATCH',
			$user,
			$password,
			self::getRequestHeaders(),
			$body
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param mixed $body
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function createDriveShareLink(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		$body
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/root/createLink");
		return HttpRequestHelper::post(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders(),
			$body
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 * @param string $spaceId
	 * @param mixed $body
	 * @param string $permissionsId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 *
	 */
	public static function setDriveLinkSharePassword(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
		string $spaceId,
		$body,
		string $permissionsId
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "drives/$spaceId/root/permissions/$permissionsId/setPassword");
		return HttpRequestHelper::post(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders(),
			$body
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getPermissionsRoleDefinitions(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
	): ResponseInterface {
		$url = self::getBetaFullUrl($baseUrl, "roleManagement/permissions/roleDefinitions");
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $permissionRole
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 *
	 * @return ResponseInterface
	 * @throws \Exception|GuzzleException
	 */
	public static function getPermissionRoleDefinition(
		string $permissionRole,
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password,
	): ResponseInterface {
		$roleId = self::getPermissionsRoleIdByName($permissionRole);
		$url = self::getBetaFullUrl($baseUrl, "roleManagement/permissions/roleDefinitions/{$roleId}");
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $requestId
	 * @param string $user
	 * @param string $password
	 * @param string $resourceId
	 * @param array $filterParams
	 *
	 * @return ResponseInterface
	 */
	public static function getActivities(
		string $baseUrl,
		string $requestId,
		string $user,
		string $password,
		string $resourceId,
		?array $filterParams = []
	): ResponseInterface {
		// 'kql=itemId' filter is required for the current implementation but it might change in future
		// See: https://github.com/owncloud/ocis/issues/9194
		$fullUrl = self::getBetaFullUrl($baseUrl, "extensions/org.libregraph/activities?kql=itemid%3A$resourceId");
		if (!empty($filterParams)) {
			foreach ($filterParams as $filter => $value) {
				$fullUrl .= "+AND+$filter%3A$value";
			}
		}
		return HttpRequestHelper::get(
			$fullUrl,
			$requestId,
			$user,
			$password
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $user
	 * @param string $password
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getFederatedUsers(
		string $baseUrl,
		string $xRequestId,
		string $user,
		string $password
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, "users?\$filter=userType eq 'Federated'");
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$user,
			$password,
			self::getRequestHeaders()
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $adminUser
	 * @param string $adminPassword
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function getAllUsers(
		string $baseUrl,
		string $xRequestId,
		string $adminUser,
		string $adminPassword,
	): ResponseInterface {
		$url = self::getFullUrl($baseUrl, "users?\$filter=userType eq 'Federated' or userType eq 'Member'");
		return HttpRequestHelper::get(
			$url,
			$xRequestId,
			$adminUser,
			$adminPassword,
			self::getRequestHeaders()
		);
	}
}
