<?php
/**
 * @author Viktor Scharf <v.scharf@opencloud.eu>
 * @copyright Copyright (c) 2025 Viktor Scharf <v.scharf@opencloud.eu>
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

use GuzzleHttp\Client;
use GuzzleHttp\Cookie\CookieJar;
use GuzzleHttp\Exception\GuzzleException;
use PHPUnit\Framework\Assert;

/**
 * Helper for obtaining bearer tokens for users
 */
class TokenHelper {
	private const LOGON_URL = '/signin/v1/identifier/_/logon';
	private const REDIRECT_URL = '/oidc-callback.html';
	private const TOKEN_URL = '/konnect/v1/token';

	// Static cache [username => token_data]
	private static array $tokenCache = [];

	/**
	 * @return bool
	 */
	public static function useBearerToken(): bool {
		return \getenv('USE_BEARER_TOKEN') === 'true';
	}

	/**
	 * Extracts base URL from a full URL
	 *
	 * @param string $url
	 *
	 * @return string the base URL
	 */
	private static function extractBaseUrl(string $url): string {
		return preg_replace('#(https?://[^/]+).*#', '$1', $url);
	}

	/**
	 * Get access and refresh tokens for a user
	 * Uses cache to avoid unnecessary requests
	 *
	 * @param string $username
	 * @param string $password
	 * @param string $url
	 *
	 * @return array ['access_token' => string, 'refresh_token' => string, 'expires_at' => int]
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getTokens(string $username, string $password, string $url): array {
		// Extract base URL. I need to send $url to get correct server in case of multiple servers (ocm suite)
		$baseUrl = self::extractBaseUrl($url);
		$cacheKey = $username . '|' . $baseUrl;

		// Check cache
		if (isset(self::$tokenCache[$cacheKey])) {
			$cachedToken = self::$tokenCache[$cacheKey];

			// Check if access token has expired
			if (time() < $cachedToken['expires_at']) {
				return $cachedToken;
			}

			$refreshedToken = self::refreshToken($cachedToken['refresh_token'], $baseUrl);
			$tokenData = [
				'access_token' => $refreshedToken['access_token'],
				'refresh_token' => $refreshedToken['refresh_token'],
				// set expiry to 240 (4 minutes) seconds to allow for some buffer
				// token actually expires in 300 seconds (5 minutes)
				'expires_at' => time() + 240
			];
			self::$tokenCache[$cacheKey] = $tokenData;
			return $tokenData;
		}

		// Get new tokens
		$cookieJar = new CookieJar();

		$continueUrl = self::getAuthorizedEndPoint($username, $password, $baseUrl, $cookieJar);
		$code = self::getCode($continueUrl, $baseUrl, $cookieJar);
		$tokens = self::getToken($code, $baseUrl, $cookieJar);

		$tokenData = [
			'access_token' => $tokens['access_token'],
			'refresh_token' => $tokens['refresh_token'],
			// set expiry to 240 (4 minutes) seconds to allow for some buffer
			// token actually expires in 300 seconds (5 minutes)
			'expires_at' => time() + 240
		];

		// Save to cache
		self::$tokenCache[$cacheKey] = $tokenData;

		return $tokenData;
	}

	/**
	 * Refresh token
	 *
	 * @param string $refreshToken
	 * @param string $baseUrl
	 *
	 * @return array
	 * @throws GuzzleException
	 * @throws Exception
	 */
	private static function refreshToken(string $refreshToken, string $baseUrl): array {
		$client = new Client(
			[
			'verify' => false,
			'http_errors' => false,
			'allow_redirects' => false
			]
		);

		$response = $client->post(
			$baseUrl . self::TOKEN_URL,
			[
			'form_params' => [
				'client_id' => 'web',
				'refresh_token' => $refreshToken,
				'grant_type' => 'refresh_token'
			]
			]
		);

		Assert::assertEquals(
			200,
			$response->getStatusCode(),
			'Token refresh failed: Expected status code 200 but received ' . $response->getStatusCode()
		);
		$data = json_decode($response->getBody()->getContents(), true);

		Assert::assertArrayHasKey('access_token', $data, 'Missing access_token in refresh response');
		Assert::assertArrayHasKey('refresh_token', $data, 'Missing refresh_token in refresh response');

		return [
			'access_token' => $data['access_token'],
			'refresh_token' => $data['refresh_token']
		];
	}

	/**
	 * Clear cached tokens for a specific user
	 *
	 * @param string $username
	 * @param string $url
	 *
	 * @return void
	 */
	public static function clearUserTokens(string $username, string $url): void {
		$baseUrl = self::extractBaseUrl($url);
		$cacheKey = $username . '|' . $baseUrl;
		unset(self::$tokenCache[$cacheKey]);
	}

	/**
	 * Clear all cached tokens
	 *
	 * @return void
	 */
	public static function clearAllTokens(): void {
		self::$tokenCache = [];
	}

	/**
	 * @param string $username
	 * @param string $password
	 * @param string $baseUrl
	 * @param CookieJar $cookieJar
	 *
	 * @return \Psr\Http\Message\ResponseInterface
	 * @throws GuzzleException
	 */
	public static function makeLoginRequest(
		string $username,
		string $password,
		string $baseUrl,
		CookieJar $cookieJar
	): \Psr\Http\Message\ResponseInterface {
		$client = new Client(
			[
			'verify' => false,
			'http_errors' => false,
			'allow_redirects' => false,
			'cookies' => $cookieJar
			]
		);

		return $client->post(
			$baseUrl . self::LOGON_URL,
			[
			'headers' => [
				'Kopano-Konnect-XSRF' => '1',
				'Referer' => $baseUrl,
				'Content-Type' => 'application/json'
			],
			'json' => [
				'params' => [$username, $password, '1'],
				'hello' => [
					'scope' => 'openid profile offline_access email',
					'client_id' => 'web',
					'redirect_uri' => $baseUrl . self::REDIRECT_URL,
					'flow' => 'oidc'
				]
			]
			]
		);
	}

	/**
	 * Step 1: Login and get continue_uri
	 *
	 * @param string $username
	 * @param string $password
	 * @param string $baseUrl
	 * @param CookieJar $cookieJar
	 *
	 * @return string
	 * @throws GuzzleException
	 * @throws Exception
	 */
	private static function getAuthorizedEndPoint(
		string $username,
		string $password,
		string $baseUrl,
		CookieJar $cookieJar
	): string {
		$response = self::makeLoginRequest($username, $password, $baseUrl, $cookieJar);

		Assert::assertEquals(
			200,
			$response->getStatusCode(),
			'Logon failed: Expected status code 200 but received: ' . $response->getStatusCode()
		);
		$data = json_decode($response->getBody()->getContents(), true);

		Assert::assertArrayHasKey(
			'hello',
			$data,
			'Logon response does not contain "hello" object'
		);

		Assert::assertArrayHasKey(
			'continue_uri',
			$data['hello'],
			'Missing continue_uri in logon response'
		);

		return $data['hello']['continue_uri'];
	}

	/**
	 * Step 2: Authorization and get code
	 *
	 * @param string $continueUrl
	 * @param string $baseUrl
	 * @param CookieJar $cookieJar
	 *
	 * @return string
	 * @throws GuzzleException
	 * @throws Exception
	 */
	private static function getCode(string $continueUrl, string $baseUrl, CookieJar $cookieJar): string {
		$client = new Client(
			[
			'verify' => false,
			'http_errors' => false,
			'allow_redirects' => false, // Disable automatic redirects
			'cookies' => $cookieJar
			]
		);

		$params = [
			'client_id' => 'web',
			'prompt' => 'none',
			'redirect_uri' => $baseUrl . self::REDIRECT_URL,
			'response_mode' => 'query',
			'response_type' => 'code',
			'scope' => 'openid profile offline_access email'
		];

		$response = $client->get(
			$continueUrl,
			[
			'query' => $params
			]
		);

		Assert::assertEquals(
			302,
			$response->getStatusCode(),
			'Authorization request failed: Expected status code 302 but received: ' . $response->getStatusCode()
		);

		$location = $response->getHeader('Location')[0] ?? '';
		Assert::assertNotEmpty($location, 'Missing Location header in authorization response');

		parse_str(parse_url($location, PHP_URL_QUERY), $queryParams);
		Assert::assertArrayHasKey('code', $queryParams, 'Missing code parameter in redirect URL');
		return $queryParams['code'];
	}

	/**
	 * Step 3: Get token
	 *
	 * @param string $code
	 * @param string $baseUrl
	 * @param CookieJar $cookieJar
	 *
	 * @return array
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 *
	 */
	private static function getToken(string $code, string $baseUrl, CookieJar $cookieJar): array {
		$client = new Client(
			[
			'verify' => false,
			'http_errors' => false,
			'allow_redirects' => false,
			'cookies' => $cookieJar
			]
		);

		$response = $client->post(
			$baseUrl . self::TOKEN_URL,
			[
			'form_params' => [
				'client_id' => 'web',
				'code' => $code,
				'redirect_uri' => $baseUrl . self::REDIRECT_URL,
				'grant_type' => 'authorization_code'
			]
			]
		);

		Assert::assertEquals(
			200,
			$response->getStatusCode(),
			'Token request failed: Expected status code 200 but received: ' . $response->getStatusCode()
		);

		$data = json_decode($response->getBody()->getContents(), true);
		Assert::assertArrayHasKey('access_token', $data, 'Missing access_token in token response');
		Assert::assertArrayHasKey('refresh_token', $data, 'Missing refresh_token in token response');

		return [
			'access_token' => $data['access_token'],
			'refresh_token' => $data['refresh_token']
		];
	}
}
