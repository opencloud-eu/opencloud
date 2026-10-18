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
 * a link and sends it back.
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
	 * guest session cookie of each guest
	 *
	 * @var array<string, string>
	 */
	private array $guestSessions = [];

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
	 * Remembers the guest link contained in an email
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
		$body = EmailHelper::getBodyOfLastEmail($guest, $this->featureContext->getStepLineRef());
		$this->rememberCredentialsFromEmail($guest, $body);
		return $body;
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
		$body = $this->readLastEmail($guest);
		Assert::assertMatchesRegularExpression(
			GuestLinkHelper::TOKEN_IN_EMAIL_REGEX,
			$body,
			"The last email of guest '$guest' does not contain a guest link"
		);
		\preg_match(GuestLinkHelper::TOKEN_IN_EMAIL_REGEX, $body, $matches);
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
	 * Waits for the next email of the guest and remembers its link without using it
	 *
	 * @param string $guest
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Given('guest :guest has received a new guest link')]
	public function guestHasReceivedANewGuestLink(string $guest): void {
		$body = $this->readNewEmail($guest);
		Assert::assertMatchesRegularExpression(
			GuestLinkHelper::TOKEN_IN_EMAIL_REGEX,
			$body,
			"The new email of guest '$guest' does not contain a guest link"
		);
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
	public function guestDownloadsSharedFile(string $guest, string $resource, string $space, string $owner): void {
		$fileId = $this->spacesContext->getResourceId($owner, $space, $resource);
		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$this->featureContext->getBaseUrl() . "/" . WebDavHelper::prefixRemotePhp("dav/spaces/$fileId"),
				$this->featureContext->getStepLineRef(),
				null,
				null,
				$this->getSessionHeaders($guest),
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
			HttpRequestHelper::sendRequest(
				$this->featureContext->getBaseUrl() . "/" . WebDavHelper::prefixRemotePhp("dav/spaces/$fileId"),
				$this->featureContext->getStepLineRef(),
				'PUT',
				null,
				null,
				$this->getSessionHeaders($guest),
				$content,
			)
		);
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
	 * The guest link token is random, so it is compared as the placeholder %guest_link_token%
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
		Assert::assertStringContainsString(
			$expectedContent,
			$actualContent,
			"The new email of guest '$guest' was expected to contain '$expectedContent'"
			. " but the received email is '$actualContent'"
		);
	}

}
