<?php declare(strict_types=1);
/**
 * @author Artur Neumann <artur@jankaritech.com>
 * @copyright Copyright (c) 2017 Artur Neumann artur@jankaritech.com
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
use GuzzleHttp\Client;
use GuzzleHttp\Exception\GuzzleException;
use InvalidArgumentException;
use PHPUnit\Framework\Assert;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\StreamInterface;
use DateTime;

/**
 * Helper to make WebDav Requests
 *
 * @author Artur Neumann <artur@jankaritech.com>
 *
 */
class WebDavHelper {
	public const DAV_VERSION_OLD = 1;
	public const DAV_VERSION_NEW = 2;
	public const DAV_VERSION_SPACES = 3;

	/**
	 * @var array of users with their different space ids
	 */
	public static array $spacesIdRef = [];

	/**
	 * @return bool
	 */
	public static function withRemotePhp(): bool {
		// use remote.php by default
		return \getenv("WITH_REMOTE_PHP") !== "false";
	}

	/**
	 * @param string $urlPath
	 *
	 * @return string
	 */
	public static function prefixRemotePhp(string $urlPath): string {
		if (self::withRemotePhp()) {
			return "remote.php/$urlPath";
		}
		return $urlPath;
	}

	/**
	 * @param string $url
	 *
	 * @return bool
	 */
	public static function isDAVRequest(string $url): bool {
		$found = \preg_match("/(\bwebdav\b|\bdav\b)/", $url);
		return (bool)$found;
	}

	/**
	 * clear space id reference for user
	 *
	 * @param string|null $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public static function removeSpaceIdReferenceForUser(
		?string $user
	): void {
		if (\array_key_exists($user, self::$spacesIdRef)) {
			unset(self::$spacesIdRef[$user]);
		}
	}

	/**
	 * @param string $namespaceString
	 *
	 * @return object
	 */
	public static function parseNamespace(string $namespaceString): object {
		//calculate the namespace prefix and namespace
		$matches = [];
		\preg_match("/^(.*)='(.*)'$/", $namespaceString, $matches);
		return (object)["namespace" => $matches[2], "prefix" => $matches[1]];
	}

	/**
	 * returns the id of a file
	 *
	 * @param string|null $baseUrl
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $path
	 * @param string|null $spaceId
	 * @param string|null $xRequestId
	 * @param int|null $davPathVersionToUse
	 *
	 * @return string
	 * @throws Exception|GuzzleException
	 */
	public static function getFileIdForPath(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $path,
		?string $spaceId = null,
		?string $xRequestId = '',
		?int $davPathVersionToUse = self::DAV_VERSION_NEW
	): string {
		$body
			= '<?xml version="1.0"?>
				<d:propfind  xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
 					<d:prop>
    					<oc:fileid />
  					</d:prop>
				</d:propfind>';
		$response = self::makeDavRequest(
			$baseUrl,
			$user,
			$password,
			"PROPFIND",
			$path,
			null,
			$spaceId,
			$xRequestId,
			$body,
			$davPathVersionToUse
		);
		\preg_match(
			'/\<oc:fileid\>([^\<]*)\<\/oc:fileid\>/',
			$response->getBody()->getContents(),
			$matches
		);

		if (!isset($matches[1])) {
			throw new Exception("could not find fileId of $path");
		}

		return $matches[1];
	}

	/**
	 * returns body for propfind
	 *
	 * @param array|null $properties
	 *
	 * @return string
	 * @throws Exception
	 */
	public static function getBodyForPropfind(?array $properties): string {
		$propertyBody = "";
		$extraNamespaces = "";
		foreach ($properties as $namespaceString => $property) {
			if (\is_int($namespaceString)) {
				//default namespace prefix if the property has no array key
				//also used if no prefix is given in the property value
				$namespacePrefix = null;
			} else {
				$ns = self::parseNamespace($namespaceString);
				$namespacePrefix = $ns->prefix;
				$extraNamespaces .= " xmlns:$namespacePrefix=\"$ns->namespace\" ";
			}
			//if a namespace prefix is given in the property value use that
			if (\strpos($property, ":") !== false) {
				$propertyParts = \explode(":", $property);
				$namespacePrefix = $propertyParts[0];
				$property = $propertyParts[1];
			}

			if ($namespacePrefix) {
				$propertyBody .= "<$namespacePrefix:$property/>";
			} else {
				$propertyBody .= "<$property/>";
			}
		}
		$body = "<?xml version=\"1.0\"?>
				<d:propfind
				   xmlns:d=\"DAV:\"
				   xmlns:oc=\"http://owncloud.org/ns\"
				   xmlns:ocs=\"http://open-collaboration-services.org/ns\"
				   $extraNamespaces>
				    <d:prop>$propertyBody</d:prop>
				</d:propfind>";
		return $body;
	}

	/**
	 * sends a PROPFIND request
	 * with these registered namespaces:
	 *  | prefix | namespace                                 |
	 *  | d      | DAV:                                      |
	 *  | oc     | http://owncloud.org/ns                    |
	 *  | ocs    | http://open-collaboration-services.org/ns |
	 *
	 * @param string|null $baseUrl
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $path
	 * @param string[] $properties
	 *        string can contain namespace prefix,
	 *        if no prefix is given 'd:' is used as prefix
	 *        if an associative array is used, then the key will be used as namespace
	 * @param string|null $xRequestId
	 * @param string|null $folderDepth
	 * @param string|null $spaceId
	 * @param string|null $type
	 * @param int|null $davPathVersionToUse
	 * @param string|null $doDavRequestAsUser
	 * @param array|null $headers
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public static function propfind(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $path,
		?array $properties,
		?string $xRequestId = '',
		?string $folderDepth = '1',
		?string $spaceId = null,
		?string $type = "files",
		?int $davPathVersionToUse = self::DAV_VERSION_NEW,
		?string $doDavRequestAsUser = null,
		?array $headers = []
	): ResponseInterface {
		$body = self::getBodyForPropfind($properties);
		$folderDepth = (string) $folderDepth;
		if ($folderDepth !== '0' && $folderDepth !== '1' && $folderDepth !== 'infinity') {
			if ($folderDepth !== '') {
				throw new InvalidArgumentException('Invalid depth value ' . $folderDepth);
			}
			$folderDepth = '1'; // OpenCloud server's default value
		}
		$headers['Depth'] = $folderDepth;
		return self::makeDavRequest(
			$baseUrl,
			$user,
			$password,
			"PROPFIND",
			$path,
			$headers,
			$spaceId,
			$xRequestId,
			$body,
			$davPathVersionToUse,
			$type,
			null,
			null,
			false,
			null,
			null,
			[],
			$doDavRequestAsUser
		);
	}

	/**
	 *
	 * @param string|null $baseUrl
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $path
	 * @param string|null $propertyName
	 * @param string|null $propertyValue
	 * @param string|null $xRequestId
	 * @param string|null $namespaceString string containing prefix and namespace
	 *                                     e.g "x1='http://whatever.org/ns'"
	 * @param int|null $davPathVersionToUse
	 * @param string|null $type
	 * @param string|null $spaceId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function proppatch(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $path,
		?string $propertyName,
		?string $propertyValue,
		?string $xRequestId = '',
		?string $namespaceString = null,
		?int $davPathVersionToUse = self::DAV_VERSION_NEW,
		?string $type="files",
		?string $spaceId = null,
	): ResponseInterface {
		if ($namespaceString !== null) {
			$ns = self::parseNamespace($namespaceString);
			$propertyBody = "<$ns->prefix:$propertyName" .
				" xmlns:$ns->prefix=\"$ns->namespace\">" .
				"$propertyValue" .
				"</$ns->prefix:$propertyName>";
		} else {
			$propertyBody = "<$propertyName>$propertyValue</$propertyName>";
		}
		$body = "<?xml version=\"1.0\"?>
				<d:propertyupdate xmlns:d=\"DAV:\"
				   xmlns:oc=\"http://owncloud.org/ns\">
				 <d:set>
				  <d:prop>$propertyBody</d:prop>
				 </d:set>
				</d:propertyupdate>";
		return self::makeDavRequest(
			$baseUrl,
			$user,
			$password,
			"PROPPATCH",
			$path,
			[],
			$spaceId,
			$xRequestId,
			$body,
			$davPathVersionToUse,
			$type
		);
	}

	/**
	 * gets namespace-prefix, namespace url and propName from provided namespaceString or property
	 * or otherwise use default
	 *
	 * @param string $namespaceString
	 * @param string $property
	 *
	 * @return array
	 */
	public static function getPropertyWithNamespaceInfo(string $namespaceString = "", string $property = ""): array {
		$namespace = "";
		$namespacePrefix = "";
		if (\is_int($namespaceString)) {
			//default namespace prefix if the property has no array key
			//also used if no prefix is given in the property value
			$namespacePrefix = "d";
			$namespace = "DAV:";
		} elseif ($namespaceString) {
			$ns = self::parseNamespace($namespaceString);
			$namespacePrefix = $ns->prefix;
			$namespace = $ns->namespace;
		}
		//if a namespace prefix is given in the property value use that
		if ($property && \strpos($property, ":")) {
			$propertyParts = \explode(":", $property);
			$namespacePrefix = $propertyParts[0];
			$property = $propertyParts[1];
		}
		return [$namespacePrefix, $namespace, $property];
	}

	/**
	 * sends HTTP request PROPPATCH method with multiple properties
	 *
	 * @param string|null $baseUrl
	 * @param string|null $user
	 * @param string|null $password
	 * @param string $path
	 * @param array|null $propertiesArray
	 * @param string|null $xRequestId
	 * @param int|null $davPathVersion
	 * @param string|null $namespaceString
	 * @param string|null $type
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function proppatchWithMultipleProps(
		?string  $baseUrl,
		?string  $user,
		?string $password,
		string  $path,
		?array $propertiesArray,
		?string $xRequestId = '',
		?int  $davPathVersion = null,
		?string  $namespaceString = null,
		?string  $type="files"
	): ResponseInterface {
		$propertyBody = "";
		foreach ($propertiesArray as $propertyArray) {
			$property = $propertyArray["propertyName"];
			$value = $propertyArray["propertyValue"];

			if ($namespaceString !== null) {
				$matches = [];
				[$namespacePrefix, $namespace, $property] = self::getPropertyWithNamespaceInfo(
					$namespaceString,
					$property
				);
				$propertyBody .= "\n\t<$namespacePrefix:$property>" .
					"$value" .
					"</$namespacePrefix:$property>";
			} else {
				$propertyBody .= "<$property>$value</$property>";
			}
		}
		$body = "<?xml version=\"1.0\"?>
				<d:propertyupdate xmlns:d=\"DAV:\"
				   xmlns:oc=\"http://owncloud.org/ns\">
				 <d:set>
				  <d:prop>$propertyBody
				  </d:prop>
				 </d:set>
				</d:propertyupdate>";
		return self::makeDavRequest(
			$baseUrl,
			$user,
			$password,
			"PROPPATCH",
			$path,
			[],
			null,
			$xRequestId,
			$body,
			$davPathVersion,
			$type
		);
	}

	/**
	 * returns the response to listing a folder (collection)
	 *
	 * @param string|null $baseUrl
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $path
	 * @param string|null $folderDepth
	 * @param string|null $spaceId
	 * @param string|null $xRequestId
	 * @param string[] $properties
	 * @param string|null $type
	 * @param int|null $davPathVersionToUse
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function listFolder(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $path,
		?string $folderDepth,
		?string $spaceId = null,
		?string $xRequestId = '',
		?array $properties = null,
		?string $type = "files",
		?int $davPathVersionToUse = self::DAV_VERSION_NEW
	): ResponseInterface {
		if (!$properties) {
			$properties = [
				'd:getetag', 'd:resourcetype'
			];
		}
		return self::propfind(
			$baseUrl,
			$user,
			$password,
			$path,
			$properties,
			$xRequestId,
			$folderDepth,
			$spaceId,
			$type,
			$davPathVersionToUse
		);
	}

	/**
	 * Generates UUIDv4
	 * Example: 123e4567-e89b-12d3-a456-426614174000
	 *
	 * @return string
	 * @throws Exception
	 */
	public static function generateUUIDv4(): string {
		// generate 16 bytes (128 bits) of random data or use the data passed into the function.
		$data = random_bytes(16);
		\assert(\strlen($data) == 16);

		$data[6] = \chr(\ord($data[6]) & 0x0f | 0x40); // set version to 0100
		$data[8] = \chr(\ord($data[8]) & 0x3f | 0x80); // set bits 6-7 to 10

		return vsprintf('%s%s-%s-%s-%s-%s%s%s', str_split(bin2hex($data), 4));
	}

	/**
	 * fetches personal space id for provided user
	 *
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $xRequestId
	 *
	 * @return string
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getPersonalSpaceIdForUser(
		string $baseUrl,
		string $user,
		string $password,
		string $xRequestId
	): string {
		if (\array_key_exists($user, self::$spacesIdRef) && \array_key_exists("personal", self::$spacesIdRef[$user])) {
			return self::$spacesIdRef[$user]["personal"];
		}

		$personalSpaceId = '';
		if (!OcHelper::isTestingOnReva()) {
			$response = GraphHelper::getMySpaces($baseUrl, $user, $password, '', $xRequestId);
			Assert::assertEquals(200, $response->getStatusCode(), "Cannot list drives for user '$user'");

			$drives = HttpRequestHelper::getJsonDecodedResponseBodyContent($response);
			foreach ($drives->value as $drive) {
				if ($drive->driveType === "personal") {
					$personalSpaceId = $drive->id;
					break;
				}
			}
		}

		if (!$personalSpaceId) {
			// the graph endpoint did not give a useful answer
			// try getting the information from the webdav endpoint
			$fullUrl = "$baseUrl/" . self::getDavPath(self::DAV_VERSION_NEW, $user);
			$response = HttpRequestHelper::sendRequest(
				$fullUrl,
				$xRequestId,
				'PROPFIND',
				$user,
				$password
			);
			Assert::assertEquals(
				207,
				$response->getStatusCode(),
				"PROPFIND for user '$user' failed so the personal space id cannot be discovered"
			);

			$responseXmlObject = HttpRequestHelper::getResponseXml(
				$response,
				__METHOD__
			);
			$xmlPart = $responseXmlObject->xpath("/d:multistatus/d:response[1]/d:propstat/d:prop/oc:spaceid");
			Assert::assertNotEmpty(
				$xmlPart,
				"The 'oc:spaceid' for user '$user' was not found in the PROPFIND response"
			);

			$personalSpaceId = $xmlPart[0]->__toString();
		}

		Assert::assertNotEmpty($personalSpaceId, "The personal space id for user '$user' was not found");

		self::$spacesIdRef[$user] = [];
		self::$spacesIdRef[$user]["personal"] = $personalSpaceId;
		return $personalSpaceId;
	}

	/**
	 * First checks if a user exists to return its space ID
	 * In case of any exception, it returns a fake space ID
	 *
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $xRequestId
	 *
	 * @return string
	 * @throws Exception|GuzzleException
	 */
	public static function getPersonalSpaceIdForUserOrFakeIfNotFound(
		string $baseUrl,
		string $user,
		string $password,
		string $xRequestId
	): string {
		if (\str_starts_with($user, "non-exist") || \str_starts_with($user, "nonexist")) {
			return self::generateUUIDv4();
		}

		return self::getPersonalSpaceIdForUser(
			$baseUrl,
			$user,
			$password,
			$xRequestId,
		);
	}

	/**
	 * sends a DAV request
	 *
	 * @param string|null $baseUrl URL of OpenCloud e.g. http://localhost:8080
	 * @param string|null $user
	 * @param string|null $password or token when bearer auth is used
	 * @param string|null $method PUT, GET, DELETE, etc.
	 * @param string|null $path
	 * @param array|null $headers
	 * @param string|null $spaceId
	 * @param string|null $xRequestId
	 * @param string|null|resource|StreamInterface $body
	 * @param int|null $davPathVersionToUse (1|2|3)
	 * @param string|null $type of request
	 * @param string|null $sourceIpAddress to initiate the request from
	 * @param string|null $authType basic|bearer
	 * @param bool $stream Set to true to stream a response rather
	 *                     than download it all up-front.
	 * @param int|null $timeout
	 * @param Client|null $client
	 * @param array|null $urlParameter to concatenate with path
	 * @param string|null $doDavRequestAsUser run the DAV as this user, if null it is the same as $user
	 * @param bool $isGivenStep is set to true if makeDavRequest is called from a "given" step
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function makeDavRequest(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $method,
		?string $path,
		?array $headers,
		?string $spaceId = null,
		?string $xRequestId = '',
		$body = null,
		?int $davPathVersionToUse = self::DAV_VERSION_OLD,
		?string $type = "files",
		?string $sourceIpAddress = null,
		?string $authType = "basic",
		?bool $stream = false,
		?int $timeout = 0,
		?Client $client = null,
		?array $urlParameter = [],
		?string $doDavRequestAsUser = null,
		?bool $isGivenStep = false,
	): ResponseInterface {
		$baseUrl = self::sanitizeUrl($baseUrl, true);

		// We need to manipulate and use path as a string.
		// So ensure that it is a string to avoid any type-conversion errors.
		if ($path === null) {
			$path = "";
		}

		// get space id if testing with spaces dav
		if ($spaceId === null && $davPathVersionToUse === self::DAV_VERSION_SPACES
			&& !\in_array($type, ["public-files", "versions"])
		) {
			$path = \ltrim($path, "/");
			if (\str_starts_with($path, "Shares/")) {
				$spaceId = GraphHelper::SHARES_SPACE_ID;
				$path = "/" . preg_replace("/^Shares\//", "", $path);
			} else {
				$spaceId = self::getPersonalSpaceIdForUserOrFakeIfNotFound(
					$baseUrl,
					$user,
					$password,
					$xRequestId
				);
			}
		}

		$suffixPath = $user;
		if ($davPathVersionToUse === self::DAV_VERSION_SPACES
			&& !\in_array($type, ["archive", "versions", "public-files"])
		) {
			$suffixPath = $spaceId;
		} elseif ($type === "versions") {
			// $path is file-id in case of versions
			$suffixPath = $path;
		}

		$davPath = self::getDavPath($davPathVersionToUse, $suffixPath, $type);

		//replace %, # and ? and in the path, Guzzle will not encode them
		$urlSpecialChar = [['%', '#', '?'], ['%25', '%23', '%3F']];
		$path = \str_replace($urlSpecialChar[0], $urlSpecialChar[1], $path);

		if (!empty($urlParameter)) {
			$urlParameter = \http_build_query($urlParameter, '', '&');
			$path .= '?' . $urlParameter;
		}
		$fullUrl = self::sanitizeUrl("$baseUrl/$davPath");
		// NOTE: no need to append path for archive and versions endpoints
		if (!\in_array($type, ["archive", "versions"])) {
			$fullUrl .= "/" . \ltrim($path, "/");
		}

		if ($authType === 'bearer') {
			$headers['Authorization'] = 'Bearer ' . $password;
			$user = null;
			$password = null;
		}
		if ($type === "public-files") {
			if ($password === null || $password === "") {
				$user = null;
			} else {
				$user = "public";
			}
		}
		$config = null;
		if ($sourceIpAddress !== null) {
			$config = [ 'curl' => [ CURLOPT_INTERFACE => $sourceIpAddress ]];
		}

		if ($headers !== null) {
			foreach ($headers as $key => $value) {
				//? and # need to be encoded in the Destination URL
				if ($key === "Destination") {
					$headers[$key] = \str_replace(
						$urlSpecialChar[0],
						$urlSpecialChar[1],
						$value
					);
					break;
				}
			}
		}

		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$xRequestId,
			$method,
			$doDavRequestAsUser ?? $user,
			$password,
			$headers,
			$body,
			$config,
			null,
			$stream,
			$timeout,
			$client,
			$isGivenStep
		);
	}

	/**
	 * get the dav path
	 *
	 * @param int $davPathVersion (1|2|3)
	 * @param string|null $userOrItemIdOrSpaceIdOrToken 'user' or 'file-id' or 'space-id' or 'public-token'
	 * @param string|null $type
	 *
	 * @return string
	 */
	public static function getDavPath(
		int $davPathVersion,
		?string $userOrItemIdOrSpaceIdOrToken = null,
		?string $type = "files"
	): string {
		switch ($type) {
			case 'archive':
				return self::prefixRemotePhp("dav/archive/$userOrItemIdOrSpaceIdOrToken/files");
			case 'versions':
				return self::prefixRemotePhp("dav/meta/$userOrItemIdOrSpaceIdOrToken/v");
			case 'comments':
				return self::prefixRemotePhp("dav/comments/files");
			default:
				break;
		}

		if ($davPathVersion === self::DAV_VERSION_SPACES) {
			if ($type === "trash-bin") {
				if ($userOrItemIdOrSpaceIdOrToken === null) {
					throw new InvalidArgumentException("Space ID is required for trash-bin endpoint");
				}
				return self::prefixRemotePhp("dav/spaces/trash-bin/$userOrItemIdOrSpaceIdOrToken");
			} elseif ($type === "public-files") {
				// spaces DAV path doesn't have own public-files endpoint
				return self::prefixRemotePhp("dav/public-files/$userOrItemIdOrSpaceIdOrToken");
			}
			// return spaces root path if spaceid is null
			// REPORT request uses spaces root path
			if ($userOrItemIdOrSpaceIdOrToken === null) {
				return self::prefixRemotePhp("dav/spaces");
			}
			return self::prefixRemotePhp("dav/spaces/$userOrItemIdOrSpaceIdOrToken");
		} else {
			if ($type === "trash-bin") {
				// Since there is no trash bin endpoint for old dav version,
				// new dav version's endpoint is used here.
				return self::prefixRemotePhp("dav/trash-bin/$userOrItemIdOrSpaceIdOrToken");
			}
			if ($davPathVersion === self::DAV_VERSION_OLD) {
				return self::prefixRemotePhp("webdav");
			} elseif ($davPathVersion === self::DAV_VERSION_NEW) {
				if ($type === "files") {
					return self::prefixRemotePhp("dav/files/$userOrItemIdOrSpaceIdOrToken");
				} elseif ($type === "public-files") {
					return self::prefixRemotePhp("dav/public-files/$userOrItemIdOrSpaceIdOrToken");
				}
				return self::prefixRemotePhp("dav");
			}
			throw new InvalidArgumentException("Invalid DAV path: $davPathVersion");
		}
	}

	/**
	 * make sure there are no double-slashes in the URL
	 *
	 * @param string|null $url
	 * @param bool|null $trailingSlash forces a trailing slash
	 *
	 * @return string
	 */
	public static function sanitizeUrl(?string $url, ?bool $trailingSlash = false): string {
		if ($trailingSlash === true) {
			$url = $url . "/";
		} else {
			$url = \rtrim($url, "/");
		}
		return \preg_replace("/([^:]\/)\/+/", '$1', $url);
	}

	/**
	 * get Mtime of File in a public link share
	 *
	 * @param string|null $baseUrl
	 * @param string|null $fileName
	 * @param string|null $token
	 * @param string|null $xRequestId
	 * @param int|null $davVersionToUse
	 *
	 * @return string
	 * @throws Exception|GuzzleException
	 */
	public static function getMtimeOfFileInPublicLinkShare(
		?string $baseUrl,
		?string $fileName,
		?string $token,
		?string $xRequestId = '',
		?int $davVersionToUse = self::DAV_VERSION_NEW
	): string {
		$response = self::propfind(
			$baseUrl,
			null,
			null,
			"$token/$fileName",
			['d:getlastmodified'],
			$xRequestId,
			'1',
			null,
			"public-files",
			$davVersionToUse
		);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$response,
			__METHOD__
		);
		$xmlPart = $responseXmlObject->xpath("//d:getlastmodified");

		return $xmlPart[0]->__toString();
	}

	/**
	 * get Mtime of a resource
	 *
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $baseUrl
	 * @param string|null $resource
	 * @param string|null $xRequestId
	 * @param int|null $davPathVersionToUse
	 * @param string|null $spaceId
	 *
	 * @return string
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public static function getMtimeOfResource(
		?string $user,
		?string $password,
		?string $baseUrl,
		?string $resource,
		?string $xRequestId = '',
		?int $davPathVersionToUse = self::DAV_VERSION_NEW,
		?string $spaceId = null,
	): string {
		$response = self::propfind(
			$baseUrl,
			$user,
			$password,
			$resource,
			["d:getlastmodified"],
			$xRequestId,
			"0",
			$spaceId,
			"files",
			$davPathVersionToUse
		);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$response,
			__METHOD__
		);
		$xmlPart = $responseXmlObject->xpath("//d:getlastmodified");
		Assert::assertArrayHasKey(
			0,
			$xmlPart,
			__METHOD__
			. " XML part does not have key 0. Expected a value at index 0 of 'xmlPart' but, found: "
			. json_encode($xmlPart)
		);
		$mtime = new DateTime($xmlPart[0]->__toString());
		return $mtime->format('U');
	}

	/**
	 * wait until the reqeust doesn't return 425 anymore
	 *
	 * @param string $url
	 * @param ?string $user
	 * @param ?string $password
	 * @param ?array $headers
	 *
	 * @return void
	 */
	public static function waitForPostProcessingToFinish(
		string $url,
		?string $user = null,
		?string $password = null,
		?array $headers = [],
	): void {
		$retried = 0;
		do {
			$response = HttpRequestHelper::sendRequest(
				$url,
				'check-425-status',
				'GET',
				$user,
				$password,
				$headers,
			);
			$statusCode = $response->getStatusCode();
			if ($statusCode !== 425) {
				return;
			}
			$tryAgain = $retried < HttpRequestHelper::numRetriesOnHttpTooEarly();
			if ($tryAgain) {
				$retried += 1;
				echo "[INFO] Waiting for post processing to finish, attempt ($retried)...\n";
				// wait 1s and try again
				\sleep(1);
			}
		} while ($tryAgain);
		echo "[ERROR] 10 seconds timeout! Post processing did not finish in time.\n";
	}
}
