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

use Behat\Behat\Context\Context;
use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use Behat\Gherkin\Node\PyStringNode;
use Behat\Step\Given;
use Behat\Step\Then;
use Behat\Step\When;
use GuzzleHttp\Exception\GuzzleException;
use PHPUnit\Framework\Assert;
use Psr\Http\Message\ResponseInterface;
use TestHelpers\BehatHelper;
use TestHelpers\EmailHelper;
use TestHelpers\GraphHelper;
use TestHelpers\GuestLinkHelper;
use TestHelpers\HttpRequestHelper;
use TestHelpers\WaitHelper;
use TestHelpers\WebDavHelper;

require_once 'bootstrap.php';

/**
 * Steps for guest links: shares with an email address that the guest opens
 * through a link (or PIN) received by email, without an OpenCloud account.
 *
 * The guest is treated like a browser: it keeps the session cookie set by verifying
 * a link or PIN and the nonce cookie set by renewing, and sends them back.
 */
class GuestLinkContext implements Context {
	private FeatureContext $featureContext;
	private SpacesContext $spacesContext;

	/**
	 * guest link tokens received by each guest, in the order they were read from the emails
	 *
	 * @var array<string, string[]>
	 */
	private array $guestLinkTokens = [];

	/**
	 * PINs received by each guest, in the order they were read from the emails
	 *
	 * @var array<string, string[]>
	 */
	private array $guestPins = [];

	/**
	 * guest session cookie of each guest
	 *
	 * @var array<string, string>
	 */
	private array $guestSessions = [];

	/**
	 * expiry of the guest session of each guest, as returned when verifying a link or PIN
	 *
	 * @var array<string, int>
	 */
	private array $guestSessionExpiry = [];

	/**
	 * nonce cookie (name => value) that binds the PIN of each guest to the "browser"
	 *
	 * @var array<string, array<string, string>>
	 */
	private array $guestNonces = [];

	/**
	 * nonce cookie (name => value) of the last successful PIN verification of each guest,
	 * the server clears the cookie then, but a client can still replay it
	 *
	 * @var array<string, array<string, string>>
	 */
	private array $usedGuestNonces = [];

	/**
	 * expiry of the PIN of each guest, taken from the max-age of the nonce cookie
	 *
	 * @var array<string, int>
	 */
	private array $guestPinExpiry = [];

	/**
	 * number of emails of each guest that the scenario has already read
	 *
	 * @var array<string, int>
	 */
	private array $readEmailCount = [];

	/**
	 * @BeforeScenario
	 *
	 * @param BeforeScenarioScope $scope
	 *
	 * @return void
	 */
	public function before(BeforeScenarioScope $scope): void {
		$environment = $scope->getEnvironment();
		$this->featureContext = BehatHelper::getContext($scope, $environment, 'FeatureContext');
		$this->spacesContext = BehatHelper::getContext($scope, $environment, 'SpacesContext');
	}

	/**
	 * Delete the emails of the guests that were invited in the scenario,
	 * so that the next scenario does not read a stale guest link.
	 *
	 * @AfterScenario
	 *
	 * @return void
	 */
	public function clearGuestMailboxes(): void {
		foreach ($this->featureContext->emailRecipients as $mailBox) {
			try {
				EmailHelper::deleteAllEmails(
					EmailHelper::getLocalEmailUrl(),
					$mailBox,
					$this->featureContext->getStepLineRef(),
				);
			} catch (Exception $e) {
				echo __METHOD__ . " could not delete emails of '$mailBox', is inbucket set up?\n" . $e->getMessage();
			}
		}
	}

	/**
	 * @param string $guest
	 * @param int $count
	 * @param int $timeoutSeconds
	 *
	 * @return array the emails of the guest, after at least $count arrived or the timeout passed
	 * @throws GuzzleException
	 */
	private function waitForEmails(string $guest, int $count, int $timeoutSeconds = 10): array {
		$this->featureContext->pushEmailRecipientAsMailBox($guest);
		$mailBox = EmailHelper::getMailBoxFromEmail($guest);
		return WaitHelper::waitUntil(
			fn () => EmailHelper::getMailBoxInformation($mailBox, $this->featureContext->getStepLineRef()),
			fn ($emails) => \count($emails) >= $count,
			500,
			$timeoutSeconds
		);
	}

	/**
	 * Remembers the guest link and the PIN contained in an email
	 *
	 * @param string $guest
	 * @param string $body
	 *
	 * @return void
	 */
	private function rememberCredentialsFromEmail(string $guest, string $body): void {
		if (\preg_match(GuestLinkHelper::TOKEN_IN_EMAIL_REGEX, $body, $matches)) {
			$knownTokens = $this->guestLinkTokens[$guest] ?? [];
			if (\end($knownTokens) !== $matches[1]) {
				$this->guestLinkTokens[$guest][] = $matches[1];
			}
		}
		if (\preg_match(GuestLinkHelper::PIN_IN_EMAIL_REGEX, $body, $matches)) {
			$knownPins = $this->guestPins[$guest] ?? [];
			if (\end($knownPins) !== $matches[1]) {
				$this->guestPins[$guest][] = $matches[1];
			}
		}
	}

	/**
	 * Waits for an email that the scenario has not read yet and returns its body
	 *
	 * @param string $guest
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	private function readNewEmail(string $guest): string {
		$expectedCount = ($this->readEmailCount[$guest] ?? 0) + 1;
		$emails = $this->waitForEmails($guest, $expectedCount);
		Assert::assertGreaterThanOrEqual(
			$expectedCount,
			\count($emails),
			"Guest '$guest' was expected to have received a new email"
		);
		$this->readEmailCount[$guest] = \count($emails);
		return $this->getBodyOfLastEmail($guest);
	}

	/**
	 * Returns the body of the last email, waiting for the first one if none was read yet
	 *
	 * @param string $guest
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	private function readLastEmail(string $guest): string {
		if (($this->readEmailCount[$guest] ?? 0) === 0) {
			return $this->readNewEmail($guest);
		}
		return $this->getBodyOfLastEmail($guest);
	}

	/**
	 * @param string $guest
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	private function getBodyOfLastEmail(string $guest): string {
		$body = EmailHelper::getBodyOfLastEmail($guest, $this->featureContext->getStepLineRef());
		$this->rememberCredentialsFromEmail($guest, $body);
		return $body;
	}

	/**
	 * @param string $guest
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	private function getGuestLinkTokenFromLastEmail(string $guest): string {
		return $this->getGuestLinkToken($guest, $this->readLastEmail($guest));
	}

	/**
	 * @param string $guest
	 * @param string $body
	 *
	 * @return string
	 */
	private function getGuestLinkToken(string $guest, string $body): string {
		Assert::assertMatchesRegularExpression(
			GuestLinkHelper::TOKEN_IN_EMAIL_REGEX,
			$body,
			"The email of guest '$guest' does not contain a guest link"
		);
		\preg_match(GuestLinkHelper::TOKEN_IN_EMAIL_REGEX, $body, $matches);
		return $matches[1];
	}

	/**
	 * @param string $guest
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	private function getPinFromLastEmail(string $guest): string {
		$body = $this->readLastEmail($guest);
		Assert::assertMatchesRegularExpression(
			GuestLinkHelper::PIN_IN_EMAIL_REGEX,
			$body,
			"The last email of guest '$guest' does not contain a PIN"
		);
		\preg_match(GuestLinkHelper::PIN_IN_EMAIL_REGEX, $body, $matches);
		return $matches[1];
	}

	/**
	 * @param string $guest
	 *
	 * @return string
	 */
	private function getGuestSession(string $guest): string {
		Assert::assertArrayHasKey(
			$guest,
			$this->guestSessions,
			"Guest '$guest' has no guest session, verify a guest link or PIN first"
		);
		return $this->guestSessions[$guest];
	}

	/**
	 * @param string $guest
	 *
	 * @return array
	 */
	private function getSessionHeaders(string $guest): array {
		return GuestLinkHelper::getHeaders([GuestLinkHelper::SESSION_COOKIE_NAME => $this->getGuestSession($guest)]);
	}

	/**
	 * Keeps the cookies that a guest endpoint sets, like a browser does
	 *
	 * @param string $guest
	 * @param ResponseInterface $response
	 *
	 * @return void
	 */
	private function rememberCookies(string $guest, ResponseInterface $response): void {
		$sessionSetCookie = GuestLinkHelper::getSessionSetCookieHeader($response);
		if ($sessionSetCookie !== null && GuestLinkHelper::getCookieValue($sessionSetCookie) !== null) {
			$this->guestSessions[$guest] = GuestLinkHelper::getCookieValue($sessionSetCookie);
			$body = \json_decode((string)$response->getBody(), true);
			if (isset($body['expirationDateTime'])) {
				$this->guestSessionExpiry[$guest] = \strtotime($body['expirationDateTime']);
			}
		}

		$nonceSetCookie = GuestLinkHelper::getNonceSetCookieHeader($response);
		if ($nonceSetCookie !== null) {
			$name = \trim(\explode('=', $nonceSetCookie, 2)[0]);
			$value = GuestLinkHelper::getCookieValue($nonceSetCookie);
			if ($value === null) {
				unset($this->guestNonces[$guest]);
			} else {
				$this->guestNonces[$guest] = [$name => $value];
				if (\preg_match('/max-age=(\d+)/i', $nonceSetCookie, $matches)) {
					$this->guestPinExpiry[$guest] = \time() + (int)$matches[1];
				}
			}
		}
	}

	/**
	 * @param string $guest
	 * @param string $token
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function verifyGuestLinkToken(string $guest, string $token): ResponseInterface {
		$response = GuestLinkHelper::verifyToken(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$token
		);
		$this->rememberCookies($guest, $response);
		return $response;
	}

	/**
	 * @param string $guest
	 * @param string $pin
	 * @param bool $withNonce
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function verifyGuestPin(string $guest, string $pin, bool $withNonce = true): ResponseInterface {
		$nonce = $withNonce ? ($this->guestNonces[$guest] ?? []) : [];
		$response = $this->sendVerifyPinRequest($pin, $nonce);
		if ($response->getStatusCode() === 200) {
			$this->usedGuestNonces[$guest] = $nonce;
		}
		$this->rememberCookies($guest, $response);
		return $response;
	}

	/**
	 * @param string $pin
	 * @param array<string, string> $nonce
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	private function sendVerifyPinRequest(string $pin, array $nonce): ResponseInterface {
		return GuestLinkHelper::verifyPin(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$this->featureContext->shareNgGetLastCreatedUserGroupShareID(),
			$pin,
			$nonce
		);
	}

	/**
	 * @param string $guest
	 * @param bool $withSession
	 * @param string|null $shareId defaults to the last created share
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function renewGuestLink(string $guest, bool $withSession, ?string $shareId = null): ResponseInterface {
		$cookies = $withSession
			? [GuestLinkHelper::SESSION_COOKIE_NAME => $this->getGuestSession($guest)]
			: [];
		$response = GuestLinkHelper::renew(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$shareId ?? $this->featureContext->shareNgGetLastCreatedUserGroupShareID(),
			$cookies
		);
		$this->rememberCookies($guest, $response);
		return $response;
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest verifies the guest link from the last email using the Graph API')]
	public function guestVerifiesTheGuestLinkFromTheLastEmail(string $guest): void {
		$token = $this->getGuestLinkTokenFromLastEmail($guest);
		$this->featureContext->setResponse($this->verifyGuestLinkToken($guest, $token));
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Given('guest :guest has verified the guest link from the last email')]
	public function guestHasVerifiedTheGuestLinkFromTheLastEmail(string $guest): void {
		$token = $this->getGuestLinkTokenFromLastEmail($guest);
		$response = $this->verifyGuestLinkToken($guest, $token);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
	}

	/**
	 * Waits for the next email of the guest and remembers its link (and PIN) without using them
	 *
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Given('guest :guest has received a new guest link')]
	public function guestHasReceivedANewGuestLink(string $guest): void {
		$this->getGuestLinkToken($guest, $this->readNewEmail($guest));
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest verifies the previous guest link using the Graph API')]
	public function guestVerifiesThePreviousGuestLink(string $guest): void {
		$this->readLastEmail($guest);
		$tokens = $this->guestLinkTokens[$guest] ?? [];
		Assert::assertGreaterThanOrEqual(2, \count($tokens), "Guest '$guest' has not received a previous guest link");
		$this->featureContext->setResponse($this->verifyGuestLinkToken($guest, $tokens[\count($tokens) - 2]));
	}

	/**
	 * @param string $guest
	 * @param string $token
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest verifies the guest link token :token using the Graph API')]
	public function guestVerifiesTheGuestLinkToken(string $guest, string $token): void {
		$this->featureContext->setResponse($this->verifyGuestLinkToken($guest, $token));
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest verifies the PIN from the last email using the Graph API')]
	public function guestVerifiesThePinFromTheLastEmail(string $guest): void {
		$pin = $this->getPinFromLastEmail($guest);
		$this->featureContext->setResponse($this->verifyGuestPin($guest, $pin));
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest verifies the PIN from the last email without the nonce cookie using the Graph API')]
	public function guestVerifiesThePinFromTheLastEmailWithoutTheNonceCookie(string $guest): void {
		$pin = $this->getPinFromLastEmail($guest);
		$this->featureContext->setResponse($this->verifyGuestPin($guest, $pin, false));
	}

	/**
	 * Replays the PIN and the nonce cookie of the last successful PIN verification,
	 * so the request fails only because they were already used
	 *
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest verifies the PIN from the last email again with the same nonce cookie using the Graph API')]
	public function guestVerifiesThePinFromTheLastEmailAgainWithTheSameNonceCookie(string $guest): void {
		Assert::assertArrayHasKey(
			$guest,
			$this->usedGuestNonces,
			"Guest '$guest' has not verified a PIN successfully"
		);
		Assert::assertNotEmpty($this->usedGuestNonces[$guest], "Guest '$guest' verified the PIN without a nonce cookie");
		$pin = $this->getPinFromLastEmail($guest);
		$response = $this->sendVerifyPinRequest($pin, $this->usedGuestNonces[$guest]);
		$this->rememberCookies($guest, $response);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Given('guest :guest has verified the PIN from the last email')]
	public function guestHasVerifiedThePinFromTheLastEmail(string $guest): void {
		$pin = $this->getPinFromLastEmail($guest);
		$response = $this->verifyGuestPin($guest, $pin);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest verifies the PIN from the previous email using the Graph API')]
	public function guestVerifiesThePinFromThePreviousEmail(string $guest): void {
		$this->readLastEmail($guest);
		$pins = $this->guestPins[$guest] ?? [];
		Assert::assertGreaterThanOrEqual(2, \count($pins), "Guest '$guest' has not received a previous PIN");
		$this->featureContext->setResponse($this->verifyGuestPin($guest, $pins[\count($pins) - 2]));
	}

	/**
	 * @param string $guest
	 * @param string $pin
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest verifies the PIN :pin using the Graph API')]
	public function guestVerifiesThePin(string $guest, string $pin): void {
		$this->featureContext->setResponse($this->verifyGuestPin($guest, $pin));
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest requests a new guest link with the guest session using the Graph API')]
	public function guestRequestsANewGuestLinkWithTheGuestSession(string $guest): void {
		$this->featureContext->setResponse($this->renewGuestLink($guest, true));
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Given('guest :guest has requested a new guest link with the guest session')]
	public function guestHasRequestedANewGuestLinkWithTheGuestSession(string $guest): void {
		$response = $this->renewGuestLink($guest, true);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
		$this->guestHasReceivedANewGuestLink($guest);
	}

	/**
	 * @param string $guest
	 * @param string $resource
	 * @param string $space
	 * @param string $owner
	 *
	 * @return string the id of the share of the resource with the guest
	 * @throws GuzzleException
	 */
	private function getGuestShareId(string $guest, string $resource, string $space, string $owner): string {
		$response = GraphHelper::getPermissionsList(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$this->featureContext->getActualUsername($owner),
			$this->featureContext->getPasswordForUser($owner),
			$this->spacesContext->getSpaceIdByName($owner, $space),
			$this->spacesContext->getResourceId($owner, $space, $resource)
		);
		foreach ($this->featureContext->getJsonDecodedResponse($response)['value'] ?? [] as $permission) {
			if (($permission['grantedToV2']['user']['id'] ?? null) === \strtolower($guest)) {
				return $permission['id'];
			}
		}
		Assert::fail("Resource '$resource' of user '$owner' is not shared with guest '$guest'");
	}

	/**
	 * @param string $guest
	 * @param string $resource
	 * @param string $space
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest requests a new guest link for resource :resource of space :space of user :owner with the guest session using the Graph API')]
	public function guestRequestsANewGuestLinkForResourceWithTheGuestSession(
		string $guest,
		string $resource,
		string $space,
		string $owner
	): void {
		$shareId = $this->getGuestShareId($guest, $resource, $space, $owner);
		$this->featureContext->setResponse($this->renewGuestLink($guest, true, $shareId));
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest requests a new guest link without a guest session using the Graph API')]
	public function guestRequestsANewGuestLinkWithoutAGuestSession(string $guest): void {
		$this->featureContext->setResponse($this->renewGuestLink($guest, false));
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Given('guest :guest has requested a new guest link without a guest session')]
	public function guestHasRequestedANewGuestLinkWithoutAGuestSession(string $guest): void {
		$response = $this->renewGuestLink($guest, false);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
		$this->guestHasReceivedANewGuestLink($guest);
	}

	/**
	 * @param string $guest
	 * @param string $pin
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Given('guest :guest has tried to verify the PIN :pin')]
	public function guestHasTriedToVerifyThePin(string $guest, string $pin): void {
		$response = $this->verifyGuestPin($guest, $pin);
		$this->featureContext->theHTTPStatusCodeShouldBe(401, "", $response);
	}

	/**
	 * Waits until the guest session has expired according to the expiry returned by the server,
	 * plus the leeway that the guestlinks auth manager of reva accepts for clock skew
	 *
	 * @param string $guest
	 *
	 * @return void
	 */
	#[Given('the guest session of guest :guest has expired')]
	public function theGuestSessionOfGuestHasExpired(string $guest): void {
		Assert::assertArrayHasKey($guest, $this->guestSessionExpiry, "Guest '$guest' has no guest session");
		$this->waitUntilTimestampHasPassed(
			$this->guestSessionExpiry[$guest] + GuestLinkHelper::SESSION_EXPIRY_LEEWAY_SECONDS
		);
	}

	/**
	 * Waits until the PIN has expired according to the max-age of the nonce cookie
	 *
	 * @param string $guest
	 *
	 * @return void
	 */
	#[Given('the PIN of guest :guest has expired')]
	public function thePinOfGuestHasExpired(string $guest): void {
		Assert::assertArrayHasKey($guest, $this->guestPinExpiry, "Guest '$guest' has not requested a PIN");
		$this->waitUntilTimestampHasPassed($this->guestPinExpiry[$guest]);
	}

	/**
	 * @param int $timestamp
	 *
	 * @return void
	 */
	private function waitUntilTimestampHasPassed(int $timestamp): void {
		$timeoutSeconds = \max(1, $timestamp - \time() + 5);
		Assert::assertLessThan(
			120,
			$timeoutSeconds,
			"Refusing to wait $timeoutSeconds seconds, configure a shorter lifetime for this scenario"
		);
		WaitHelper::waitUntil(
			fn () => \time(),
			fn ($now) => $now > $timestamp,
			500,
			$timeoutSeconds
		);
	}


	/**
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest lists the shares shared with them using the Graph API')]
	public function guestListsTheSharesSharedWithThem(string $guest): void {
		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$this->featureContext->getBaseUrl() . '/graph/v1beta1/me/drive/sharedWithMe',
				$this->featureContext->getStepLineRef(),
				null,
				null,
				$this->getSessionHeaders($guest),
			)
		);
	}

	/**
	 * @param string $guest
	 * @param string $resource
	 * @param string $space
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest downloads file :resource shared from space :space of user :owner using the WebDAV API')]
	public function guestDownloadsSharedFile(
		string $guest,
		string $resource,
		string $space,
		string $owner
	): void {
		$fileId = $this->spacesContext->getResourceId($owner, $space, $resource);
		$this->featureContext->setResponse(
			WebDavHelper::makeDavRequest(
				$this->featureContext->getBaseUrl(),
				null,
				null,
				'GET',
				'',
				\array_merge($this->getSessionHeaders($guest), []),
				$fileId,
				$this->featureContext->getStepLineRef(),
				null,
				WebDavHelper::DAV_VERSION_SPACES
			)
		);
	}

	/**
	 * @param string $guest
	 * @param string $content
	 * @param string $resource
	 * @param string $space
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest uploads content :content to file :resource shared from space :space of user :owner using the WebDAV API')]
	public function guestUploadsContentToSharedFile(
		string $guest,
		string $content,
		string $resource,
		string $space,
		string $owner
	): void {
		$fileId = $this->spacesContext->getResourceId($owner, $space, $resource);
		$this->featureContext->setResponse(
			WebDavHelper::makeDavRequest(
				$this->featureContext->getBaseUrl(),
				null,
				null,
				'PUT',
				'',
				\array_merge($this->getSessionHeaders($guest), []),
				$fileId,
				$this->featureContext->getStepLineRef(),
				$content,
				WebDavHelper::DAV_VERSION_SPACES
			)
		);
	}

	/**
	 * @param string $guest
	 * @param string $folder
	 * @param string $space
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest lists the content of folder :folder of space :space of user :owner using the WebDAV API')]
	public function guestListsTheContentOfFolder(
		string $guest,
		string $folder,
		string $space,
		string $owner
	): void {
		$folderId = $this->spacesContext->getResourceId($owner, $space, $folder);
		$this->featureContext->setResponse(
			WebDavHelper::makeDavRequest(
				$this->featureContext->getBaseUrl(),
				null,
				null,
				'PROPFIND',
				'',
				\array_merge($this->getSessionHeaders($guest), ['Depth' => '1']),
				$folderId,
				$this->featureContext->getStepLineRef(),
				null,
				WebDavHelper::DAV_VERSION_SPACES
			)
		);
	}

	/**
	 * @param string $guest
	 * @param string $content
	 * @param string $path
	 * @param string $folder
	 * @param string $space
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest uploads content :content to :path inside folder :folder of space :space of user :owner using the WebDAV API')]
	public function guestUploadsContentInsideFolder(
		string $guest,
		string $content,
		string $path,
		string $folder,
		string $space,
		string $owner
	): void {
		$folderId = $this->spacesContext->getResourceId($owner, $space, $folder);
		$this->featureContext->setResponse(
			WebDavHelper::makeDavRequest(
				$this->featureContext->getBaseUrl(),
				null,
				null,
				'PUT',
				$path,
				\array_merge($this->getSessionHeaders($guest), []),
				$folderId,
				$this->featureContext->getStepLineRef(),
				$content,
				WebDavHelper::DAV_VERSION_SPACES
			)
		);
	}

	/**
	 * @param string $guest
	 * @param string $path
	 * @param string $folder
	 * @param string $space
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest creates folder :path inside folder :folder of space :space of user :owner using the WebDAV API')]
	public function guestCreatesFolderInsideFolder(
		string $guest,
		string $path,
		string $folder,
		string $space,
		string $owner
	): void {
		$folderId = $this->spacesContext->getResourceId($owner, $space, $folder);
		$this->featureContext->setResponse(
			WebDavHelper::makeDavRequest(
				$this->featureContext->getBaseUrl(),
				null,
				null,
				'MKCOL',
				$path,
				\array_merge($this->getSessionHeaders($guest), []),
				$folderId,
				$this->featureContext->getStepLineRef(),
				null,
				WebDavHelper::DAV_VERSION_SPACES
			)
		);
	}

	/**
	 * @param string $guest
	 * @param string $path
	 * @param string $folder
	 * @param string $space
	 * @param string $owner
	 * @param string $newName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest renames :path inside folder :folder of space :space of user :owner to :newName using the WebDAV API')]
	public function guestRenamesInsideFolder(
		string $guest,
		string $path,
		string $folder,
		string $space,
		string $owner,
		string $newName
	): void {
		$folderId = $this->spacesContext->getResourceId($owner, $space, $folder);
		$this->featureContext->setResponse(
			WebDavHelper::makeDavRequest(
				$this->featureContext->getBaseUrl(),
				null,
				null,
				'MOVE',
				$path,
				\array_merge($this->getSessionHeaders($guest), ['Destination' => $this->featureContext->getBaseUrl() . '/' . WebDavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $folderId) . '/' . $newName]),
				$folderId,
				$this->featureContext->getStepLineRef(),
				null,
				WebDavHelper::DAV_VERSION_SPACES
			)
		);
	}

	/**
	 * @param string $guest
	 * @param string $path
	 * @param string $folder
	 * @param string $space
	 * @param string $owner
	 * @param string $destination
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest tries to move :path inside folder :folder of space :space of user :owner to :destination in the root of that space using the WebDAV API')]
	public function guestTriesToMoveOutOfTheSharedFolder(
		string $guest,
		string $path,
		string $folder,
		string $space,
		string $owner,
		string $destination
	): void {
		$folderId = $this->spacesContext->getResourceId($owner, $space, $folder);
		$spaceRootId = $this->spacesContext->getResourceId($owner, $space, '');
		$this->featureContext->setResponse(
			WebDavHelper::makeDavRequest(
				$this->featureContext->getBaseUrl(),
				null,
				null,
				'MOVE',
				$path,
				\array_merge($this->getSessionHeaders($guest), ['Destination' => $this->featureContext->getBaseUrl() . '/' . WebDavHelper::getDavPath(WebDavHelper::DAV_VERSION_SPACES, $spaceRootId) . '/' . $destination]),
				$folderId,
				$this->featureContext->getStepLineRef(),
				null,
				WebDavHelper::DAV_VERSION_SPACES
			)
		);
	}

	/**
	 * @param string $guest
	 * @param string $method
	 * @param string $endpoint e.g. "invite", "createLink" or "permissions"
	 * @param string $resource
	 * @param string $space
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest sends HTTP method :method to the Graph endpoint :endpoint of resource :resource of space :space of user :owner')]
	public function guestSendsHttpMethodToGraphEndpointOfResource(
		string $guest,
		string $method,
		string $endpoint,
		string $resource,
		string $space,
		string $owner
	): void {
		$spaceId = $this->spacesContext->getSpaceIdByName($owner, $space);
		$itemId = $this->spacesContext->getResourceId($owner, $space, $resource);
		$this->featureContext->setResponse(
			HttpRequestHelper::sendRequest(
				GraphHelper::getBetaFullUrl(
					$this->featureContext->getBaseUrl(),
					"drives/$spaceId/items/$itemId/$endpoint"
				),
				$this->featureContext->getStepLineRef(),
				$method,
				null,
				null,
				$this->getSessionHeaders($guest),
				$method === 'GET' ? null : '{}',
			)
		);
	}

	/**
	 * @param string $guest
	 *
	 * @return void
	 */
	#[Given('the signature of the guest session of guest :guest has been changed')]
	public function theSignatureOfTheGuestSessionOfGuestHasBeenChanged(string $guest): void {
		$session = $this->getGuestSession($guest);
		// change the first character of the signature, the last one can carry only padding bits
		$position = \strrpos($session, '.') + 1;
		$this->guestSessions[$guest] = \substr_replace($session, $session[$position] === 'A' ? 'B' : 'A', $position, 1);
	}

	/**
	 * @param string $guest
	 * @param string $method
	 * @param string $url
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('guest :guest sends HTTP method :method to URL :url with the guest session')]
	public function guestSendsHttpMethodToUrlWithTheGuestSession(string $guest, string $method, string $url): void {
		$url = $this->featureContext->substituteInLineCodes($url);
		$this->featureContext->setResponse(
			HttpRequestHelper::sendRequest(
				$this->featureContext->getBaseUrl() . $url,
				$this->featureContext->getStepLineRef(),
				$method,
				null,
				null,
				$this->getSessionHeaders($guest),
			)
		);
	}

	/**
	 * @return void
	 */
	#[Then('the response should contain a guest session cookie')]
	public function theResponseShouldContainAGuestSessionCookie(): void {
		$setCookie = GuestLinkHelper::getSessionSetCookieHeader($this->featureContext->getResponse());
		Assert::assertNotNull($setCookie, "The response does not set the guest session cookie");
		$this->assertCookieIsHardened($setCookie);
	}

	/**
	 * @return void
	 */
	#[Then('the response should not contain a guest session cookie')]
	public function theResponseShouldNotContainAGuestSessionCookie(): void {
		Assert::assertNull(
			GuestLinkHelper::getSessionSetCookieHeader($this->featureContext->getResponse()),
			"The response was not expected to set the guest session cookie"
		);
	}

	/**
	 * @return void
	 */
	#[Then('the response should contain a nonce cookie')]
	public function theResponseShouldContainANonceCookie(): void {
		$setCookie = GuestLinkHelper::getNonceSetCookieHeader($this->featureContext->getResponse());
		Assert::assertNotNull($setCookie, "The response does not set the nonce cookie");
		Assert::assertNotNull(GuestLinkHelper::getCookieValue($setCookie), "The response clears the nonce cookie");
		$this->assertCookieIsHardened($setCookie);
	}

	/**
	 * @return void
	 */
	#[Then('the response should not contain a nonce cookie')]
	public function theResponseShouldNotContainANonceCookie(): void {
		$setCookie = GuestLinkHelper::getNonceSetCookieHeader($this->featureContext->getResponse());
		Assert::assertTrue(
			$setCookie === null || GuestLinkHelper::getCookieValue($setCookie) === null,
			"The response was not expected to set the nonce cookie, but it sets '$setCookie'"
		);
	}

	/**
	 * @param string $setCookie
	 *
	 * @return void
	 */
	private function assertCookieIsHardened(string $setCookie): void {
		$attributes = \array_map('trim', \explode(';', \strtolower($setCookie)));
		foreach (['path=/', 'httponly', 'secure', 'samesite=lax'] as $expectedAttribute) {
			Assert::assertContains(
				$expectedAttribute,
				$attributes,
				"The cookie '$setCookie' is missing the attribute '$expectedAttribute'"
			);
		}
	}

	/**
	 * The guest link token and the PIN are random, so they are compared as
	 * the placeholders %guest_link_token% and %guest_pin%
	 *
	 * @param string $guest
	 * @param string $sender
	 * @param PyStringNode $content
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Then('guest :guest should have received the following email from user :sender')]
	public function guestShouldHaveReceivedTheFollowingEmailFromUser(
		string $guest,
		string $sender,
		PyStringNode $content
	): void {
		$expectedContent = $this->featureContext->substituteInLineCodes(
			\str_replace("\r\n", "\n", $content->getRaw()),
			$sender
		);
		$actualContent = $this->readNewEmail($guest);
		$actualContent = \preg_replace(GuestLinkHelper::TOKEN_IN_EMAIL_REGEX, '/g/%guest_link_token%', $actualContent);
		$actualContent = \preg_replace(GuestLinkHelper::PIN_IN_EMAIL_REGEX, 'Your PIN: %guest_pin%', $actualContent);
		Assert::assertStringContainsString(
			$expectedContent,
			$actualContent,
			"The new email of guest '$guest' was expected to contain '$expectedContent'"
			. " but the received email is '$actualContent'"
		);
	}

	/**
	 * Waits a few seconds to make sure that no further email arrives
	 *
	 * @param string $guest
	 * @param int $count
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Then('/^guest "([^"]*)" should have received (\d+) emails? in total$/')]
	public function guestShouldHaveReceivedEmailsInTotal(string $guest, int $count): void {
		$emails = $this->waitForEmails($guest, $count + 1, 5);
		Assert::assertCount($count, $emails, "Guest '$guest' was expected to have received $count email(s)");
	}
}
