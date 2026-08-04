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

/**
 * Helper for waiting on asynchronous/eventually-consistent server state.
 * @package TestHelpers
 */
class WaitHelper {
	/**
	 * Overall time to keep polling before giving up (seconds).
	 */
	public const TIMEOUT_SECONDS = 10;

	/**
	 * Pause between two attempts (milliseconds).
	 */
	public const INTERVAL_MS = 500;

	/**
	 * Repeat $makeAttempt until $shouldStop returns true or the timeout elapses.
	 *
	 * @param callable $makeAttempt  makes one attempt (e.g. sends a request) and returns its result
	 * @param callable $shouldStop   receives that result, returns true to stop polling
	 *
	 * @return mixed the last result from $makeAttempt
	 */
	public static function waitUntil(callable $makeAttempt, callable $shouldStop): mixed {
		$deadline = \microtime(true) + self::TIMEOUT_SECONDS;
		$result = $makeAttempt();
		while (!$shouldStop($result) && \microtime(true) < $deadline) {
			\usleep(self::INTERVAL_MS * 1000);
			$result = $makeAttempt();
		}
		return $result;
	}
}
