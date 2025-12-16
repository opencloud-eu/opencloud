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
			$emailPort = "9000";
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
	 * Returns general response information about the provided mailbox
	 * A mailbox is created automatically in InBucket for every unique email sender|receiver
	 *
	 * @param string $mailBox
	 * @param string|null $xRequestId
	 *
	 * @return array
	 * @throws GuzzleException
	 */
	public static function getMailBoxInformation(string $mailBox, ?string $xRequestId = null): array {
		$response = HttpRequestHelper::get(
			self::getLocalEmailUrl() . "/api/v1/mailbox/" . $mailBox,
			$xRequestId,
			null,
			null,
			['Content-Type' => 'application/json']
		);
		return \json_decode($response->getBody()->getContents());
	}

	/**
	 * returns body content of a specific email (mailBox) with email ID (mailbox Id)
	 *
	 * @param string $mailBox
	 * @param string $mailboxId
	 * @param string|null $xRequestId
	 *
	 * @return object
	 * @throws GuzzleException
	 */
	public static function getBodyOfAnEmailById(
		string $mailBox,
		string $mailboxId,
		?string $xRequestId = null
	): object {
		$response = HttpRequestHelper::get(
			self::getLocalEmailUrl() . "/api/v1/mailbox/" . $mailBox . "/" . $mailboxId,
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
			$emailId = \array_pop($emails)->id;
			$response = self::getBodyOfAnEmailById($mailBox, $emailId, $xRequestId);
			$body = \str_replace(
				"\r\n",
				"\n",
				\quoted_printable_decode($response->body->text . "\n" . $response->body->html)
			);
			return $body;
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
			$url . "/api/v1/mailbox/" . $mailBox,
			$xRequestId,
		);
	}
}
