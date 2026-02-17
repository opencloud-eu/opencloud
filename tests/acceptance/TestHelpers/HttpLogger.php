<?php declare(strict_types=1);
/**
 * @author Sajan Gurung <sajan@jankaritech.com>
 * @copyright Copyright (c) 2023, ownCloud GmbH
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

use Psr\Http\Message\RequestInterface;
use Psr\Http\Message\ResponseInterface;

/**
 * Helper for logging HTTP requests and responses
 */
class HttpLogger {
	/**
	 * @return string
	 */
	public static function getLogDir(): string {
		return __DIR__ . '/../logs';
	}

	/**
	 * @return string
	 */
	public static function getFailedLogPath(): string {
		return self::getLogDir() . "/failed.log";
	}

	/**
	 * @return string
	 */
	public static function getScenarioLogPath(): string {
		return self::getLogDir() . "/scenario.log";
	}

	/**
	 * @param string $logFile
	 * @param string $logMessage
	 *
	 * @return void
	 */
	public static function writeLog(string $logFile, string $logMessage): void {
		$file = \fopen($logFile, 'a+') or die('Cannot open file:  ' . $logFile);
		\fwrite($file, $logMessage);
		\fclose($file);
	}

	/**
	 * @param RequestInterface $request
	 *
	 * @return void
	 */
	public static function logRequest(RequestInterface $request): void {
		$method = $request->getMethod();
		$path = $request->getUri()->getPath();
		$query = $request->getUri()->getQuery();
		$body = $request->getBody();

		$headers = "";
		foreach ($request->getHeaders() as $key => $value) {
			$headers = $key . ": " . $value[0] . "\n";
		}

		$logMessage = "\t\t_______________________________________________________________________\n\n";
		$logMessage .= "\t\t==> REQUEST [" . self::getCurrentDateTime() . "]\n";
		$logMessage .= "\t\t$method $path\n";
		$logMessage .= $query ? "\t\tQUERY: $query\n" : "";
		$logMessage .= "\t\t$headers";

		if ($body->getSize() > 0) {
			$logMessage .= "\t\t==> REQ BODY\n";
			$logMessage .= "\t\t$body\n";
		}
		$logMessage .= "\n";
		self::writeLog(self::getScenarioLogPath(), $logMessage);
	}

	/**
	 * @param ResponseInterface $response
	 *
	 * @return void
	 */
	public static function logResponse(ResponseInterface $response): void {
		$statusCode = $response->getStatusCode();
		$statusMessage = $response->getReasonPhrase();
		$body = $response->getBody();
		$headers = "";

		foreach ($response->getHeaders() as $key => $value) {
			$headers = $key . ": " . $value[0] . "\n";
		}

		$logMessage = "\t\t<== RESPONSE [" . self::getCurrentDateTime() . "]\n";
		$logMessage .= "\t\t$statusCode $statusMessage\n";
		$logMessage .= "\t\t$headers";

		if ($body->getSize() > 0) {
			$logMessage .= "\t\t<== RES BODY\n";
			foreach (\explode("\n", \strval($body)) as $line) {
				$logMessage .= "\t\t$line\n";
			}
		}
		// rewind the body stream so that later code can read from the start.
		$response->getBody()->rewind();

		$logMessage = \rtrim($logMessage) . "\n\n";
		self::writeLog(self::getScenarioLogPath(), $logMessage);
	}

	/**
	 * Returns current date and time in format: 1999-01-31T23:01:59
	 *
	 * @return string
	 */
	public static function getCurrentDateTime(): string {
		return date('Y-m-d\TG:i:s');
	}
}
