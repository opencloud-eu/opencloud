<?php declare(strict_types=1);
/**
 * @author Prajwol Amatya <prajwol@jankaritech.com>
 * @copyright Copyright (c) 2023 Prajwol Amatya prajwol@jankaritech.com
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
use Psr\Http\Message\ResponseInterface;

/**
 * A helper class for managing emails
 */
class EmailHelper {
	/**
	 * @param string $emailAddress
	 *
	 * @return string
	 */
	public static function getMailBoxFromEmail(string $emailAddress): string {
		return explode("@", $emailAddress)[0];
	}

	/**
	 * Returns the host and port where Email messages can be read and deleted
	 * by the test runner.
	 *
	 * @return string
	 */
	public static function getLocalEmailUrl(): string {
		$localEmailHost = self::getLocalEmailHost();
		$emailPort = \getenv('EMAIL_PORT');
		if ($emailPort === false) {
			$emailPort = "8025";
		}
		return "http://$localEmailHost:$emailPort";
	}

	/**
	 * Returns the host name or address of the Email server as seen from the
	 * point of view of the system-under-test.
	 *
	 * @return string
	 */
	public static function getEmailHost(): string {
		$emailHost = \getenv('EMAIL_HOST');
		if ($emailHost === false) {
			$emailHost = "127.0.0.1";
		}
		return $emailHost;
	}

	/**
	 * Returns the host name or address of the Email server as seen from the
	 * point of view of the test runner.
	 *
	 * @return string
	 */
	public static function getLocalEmailHost(): string {
		$localEmailHost = \getenv('LOCAL_EMAIL_HOST');
		if ($localEmailHost === false) {
			$localEmailHost = self::getEmailHost();
		}
		return $localEmailHost;
	}

	/**
	 * Returns the emails sent to the provided mailbox (the local part of the
	 * address), the oldest first
	 *
	 * @param string $mailBox
	 * @param string|null $xRequestId
	 *
	 * @return array
	 * @throws GuzzleException
	 */
	public static function getMailBoxInformation(string $mailBox, ?string $xRequestId = null): array {
		$response = HttpRequestHelper::get(
			self::getLocalEmailUrl() . "/api/v1/search?query=" . \urlencode("to:$mailBox"),
			$xRequestId,
			null,
			null,
			['Content-Type' => 'application/json']
		);
		$emails = \json_decode($response->getBody()->getContents())->messages;
		// mailpit also finds parts of addresses (to:alice finds malice@...)
		$emails = \array_filter(
			$emails,
			fn ($email) => \in_array(
				\strtolower($mailBox),
				\array_map(fn ($to) => \strtolower(self::getMailBoxFromEmail($to->Address)), $email->To)
			)
		);
		// mailpit lists the newest first
		return \array_reverse(\array_values($emails));
	}

	/**
	 * returns a specific email by its ID
	 *
	 * @param string $mailboxId
	 * @param string|null $xRequestId
	 *
	 * @return object
	 * @throws GuzzleException
	 */
	public static function getBodyOfAnEmailById(
		string $mailboxId,
		?string $xRequestId = null
	): object {
		$response = HttpRequestHelper::get(
			self::getLocalEmailUrl() . "/api/v1/message/" . $mailboxId,
			$xRequestId,
			null,
			null,
			['Content-Type' => 'application/json']
		);
		return \json_decode($response->getBody()->getContents());
	}

	/**
	 * Returns the body of the last received email for the provided receiver
	 *
	 * @param string $emailAddress
	 * @param string $xRequestId
	 *
	 * @return string
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getBodyOfLastEmail(
		string $emailAddress,
		string $xRequestId,
	): string {
		$mailBox = self::getMailBoxFromEmail($emailAddress);
		$emails = self::getMailboxInformation($mailBox, $xRequestId);
		if (!empty($emails)) {
			$emailId = \array_pop($emails)->ID;
			$response = self::getBodyOfAnEmailById($emailId, $xRequestId);
			// mailpit returns the body decoded already
			return \str_replace("\r\n", "\n", $response->Text . "\n" . $response->HTML);
		}
		return "";
	}

	/**
	 * Deletes all the emails for the provided mailbox
	 *
	 * @param string $url
	 * @param string $mailBox
	 * @param string $xRequestId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function deleteAllEmails(
		string $url,
		string $mailBox,
		string $xRequestId,
	): ResponseInterface {
		return HttpRequestHelper::delete(
			$url . "/api/v1/search?query=" . \urlencode("to:$mailBox"),
			$xRequestId,
		);
	}
}
