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

namespace TestHelpers;

use GuzzleHttp\Exception\ConnectException;
use TestHelpers\HttpRequestHelper;
use GuzzleHttp\Exception\GuzzleException;
use GuzzleHttp\Psr7\Request;
use Psr\Http\Message\ResponseInterface;

/**
 * A helper class for configuring OpenCloud server
 */
class OcConfigHelper {
	public static $postProcessingDelay = 0;

	/**
	 * @return int
	 */
	public static function getPostProcessingDelay(): int {
		return self::$postProcessingDelay;
	}

	/**
	 * @param string $postProcessingDelay
	 *
	 * @return void
	 */
	public static function setPostProcessingDelay(string $postProcessingDelay): void {
		// extract number from string
		$delay = (int) filter_var($postProcessingDelay, FILTER_SANITIZE_NUMBER_INT);
		self::$postProcessingDelay = $delay;
	}

	/**
	 * @param string $url
	 * @param string $method
	 * @param ?string $body
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function sendRequest(
		string $url,
		string $method,
		?string $body = ""
	): ResponseInterface {
		$client = HttpRequestHelper::createClient();
		$request = new Request(
			$method,
			$url,
			[],
			$body
		);

		try {
			$response = $client->send($request);
		} catch (ConnectException $e) {
			throw new \Error(
				"Cannot connect to the ocwrapper at the moment,"
				. "make sure that ocwrapper is running before proceeding with the test run.\n"
				. $e->getMessage()
			);
		} catch (GuzzleException $ex) {
			$response = $ex->getResponse();

			if ($response === null) {
				throw $ex;
			}
		}

		return $response;
	}

	/**
	 * @return string
	 */
	public static function getWrapperUrl(): string {
		$url = \getenv("OC_WRAPPER_URL");
		if ($url === false) {
			$url = "http://localhost:5200";
		}
		return $url;
	}

	/**
	 * @param array $envs
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function reConfigureOc(array $envs): ResponseInterface {
		$url = self::getWrapperUrl() . "/config";
		return self::sendRequest($url, "PUT", \json_encode($envs));
	}

	/**
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function rollbackOc(): ResponseInterface {
		$url = self::getWrapperUrl() . "/rollback";
		return self::sendRequest($url, "DELETE");
	}

	/**
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function stopOpencloud(): ResponseInterface {
		$url = self::getWrapperUrl() . "/stop";
		return self::sendRequest($url, "POST");
	}

	/**
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function startOpencloud(): ResponseInterface {
		$url = self::getWrapperUrl() . "/start";
		return self::sendRequest($url, "POST");
	}
}
