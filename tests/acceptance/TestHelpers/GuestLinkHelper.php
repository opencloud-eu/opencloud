<?php declare(strict_types=1);
/**
 * @author Viktor Scharf OpenCloud GmbH <v.scharf@opencloud.eu>
 * @copyright Copyright (c) 2026 OpenCloud GmbH <v.scharf@opencloud.eu>
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
use Psr\Http\Message\ResponseInterface;

/**
 * Helper for the unauthenticated guest links endpoints of the auth-guest service
 */
class GuestLinkHelper {
	public const SESSION_COOKIE_NAME = '__Host-oc_guest_session';

	/**
	 * the nonce cookie that binds a PIN to the browser is named "<base name>_<share id hash>"
	 */
	public const NONCE_COOKIE_BASE_NAME = '__Host-oc_guest_nonce';

	/**
	 * the guestlinks auth manager of reva still accepts a guest session for this long after it expired
	 */
	public const SESSION_EXPIRY_LEEWAY_SECONDS = 30;

	/**
	 * matches the guest link token in a guest link email, e.g. "/g/v1.<share id hash>.<secret>"
	 */
	public const TOKEN_IN_EMAIL_REGEX = '/\/g\/(v1\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)/';

	/**
	 * matches the PIN in a renewed guest link email
	 */
	public const PIN_IN_EMAIL_REGEX = '/Your PIN: (\d{6})/';

	/**
	 * @param string $baseUrl
	 * @param string $action verify/token|verify/pin|renew
	 *
	 * @return string
	 */
	public static function getUrl(string $baseUrl, string $action): string {
		return GraphHelper::getBetaFullUrl($baseUrl, "extensions/org.libregraph/guestLinks/$action");
	}

	/**
	 * @param array<string, string> $cookies cookie name => value
	 *
	 * @return array
	 */
	public static function getHeaders(array $cookies = []): array {
		$headers = ['Content-Type' => 'application/json'];
		if ($cookies !== []) {
			$headers['Cookie'] = \implode(
				'; ',
				\array_map(fn ($name, $value) => "$name=$value", \array_keys($cookies), $cookies)
			);
		}
		return $headers;
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $token
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function verifyToken(string $baseUrl, string $xRequestId, string $token): ResponseInterface {
		return HttpRequestHelper::post(
			self::getUrl($baseUrl, 'verify/token'),
			$xRequestId,
			null,
			null,
			self::getHeaders(),
			\json_encode(['token' => $token])
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $permissionId
	 * @param string $pin
	 * @param array<string, string> $cookies the nonce cookie set by renew
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function verifyPin(
		string $baseUrl,
		string $xRequestId,
		string $permissionId,
		string $pin,
		array $cookies = []
	): ResponseInterface {
		return HttpRequestHelper::post(
			self::getUrl($baseUrl, 'verify/pin'),
			$xRequestId,
			null,
			null,
			self::getHeaders($cookies),
			\json_encode(['pin' => $pin, 'permissionId' => $permissionId])
		);
	}

	/**
	 * @param string $baseUrl
	 * @param string $xRequestId
	 * @param string $permissionId
	 * @param array<string, string> $cookies the guest session cookie, if any
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function renew(
		string $baseUrl,
		string $xRequestId,
		string $permissionId,
		array $cookies = []
	): ResponseInterface {
		return HttpRequestHelper::post(
			self::getUrl($baseUrl, 'renew'),
			$xRequestId,
			null,
			null,
			self::getHeaders($cookies),
			\json_encode(['permissionId' => $permissionId])
		);
	}

	/**
	 * Returns the Set-Cookie headers of the response, keyed by the cookie name
	 *
	 * @param ResponseInterface $response
	 *
	 * @return array<string, string>
	 */
	public static function getSetCookieHeaders(ResponseInterface $response): array {
		$setCookies = [];
		foreach ($response->getHeader('Set-Cookie') as $setCookie) {
			$name = \trim(\explode('=', \explode(';', $setCookie, 2)[0], 2)[0]);
			$setCookies[$name] = $setCookie;
		}
		return $setCookies;
	}

	/**
	 * Returns the value of a cookie set by the response, or null when it is not set or is cleared
	 *
	 * @param string $setCookie
	 *
	 * @return string|null
	 */
	public static function getCookieValue(string $setCookie): ?string {
		$value = \explode('=', \explode(';', $setCookie, 2)[0], 2)[1] ?? '';
		$attributes = \strtolower($setCookie);
		if ($value === '' || \str_contains($attributes, 'max-age=0') || \preg_match('/max-age=-\d/', $attributes)) {
			return null;
		}
		return $value;
	}

	/**
	 * Returns the raw Set-Cookie header of the guest session cookie, or null
	 *
	 * @param ResponseInterface $response
	 *
	 * @return string|null
	 */
	public static function getSessionSetCookieHeader(ResponseInterface $response): ?string {
		return self::getSetCookieHeaders($response)[self::SESSION_COOKIE_NAME] ?? null;
	}

	/**
	 * Returns the raw Set-Cookie header of the nonce cookie, or null
	 *
	 * @param ResponseInterface $response
	 *
	 * @return string|null
	 */
	public static function getNonceSetCookieHeader(ResponseInterface $response): ?string {
		foreach (self::getSetCookieHeaders($response) as $name => $setCookie) {
			if (\str_starts_with($name, self::NONCE_COOKIE_BASE_NAME)) {
				return $setCookie;
			}
		}
		return null;
	}
}
