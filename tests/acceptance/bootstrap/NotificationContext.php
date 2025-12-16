<?php declare(strict_types=1);
/**
 * @author Viktor Scharf <vscharf@owncloud.com>
 * @copyright Copyright (c) 2023 Viktor Scharf vscharf@owncloud.com
 */

use Behat\Behat\Context\Context;
use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use Behat\Gherkin\Node\TableNode;
use Behat\Gherkin\Node\PyStringNode;
use PHPUnit\Framework\Assert;
use GuzzleHttp\Exception\GuzzleException;
use Psr\Http\Message\ResponseInterface;
use TestHelpers\EmailHelper;
use TestHelpers\OcsApiHelper;
use TestHelpers\GraphHelper;
use TestHelpers\SettingsHelper;
use TestHelpers\BehatHelper;

require_once 'bootstrap.php';

/**
 * Defines application features from the specific context.
 */
class NotificationContext implements Context {
	private FeatureContext $featureContext;
	private SpacesContext $spacesContext;
	private string $notificationEndpointPath = '/apps/notifications/api/v1/notifications?format=json';
	private string $globalNotificationEndpointPath = '/apps/notifications/api/v1/notifications/global';

	private array $notificationIds;

	/**
	 * @return array[]
	 */
	public function getNotificationIds(): array {
		return $this->notificationIds;
	}

	/**
	 * @return array[]
	 */
	public function getLastNotificationId(): array {
		return \end($this->notificationIds);
	}

	/**
	 * @AfterScenario
	 *
	 * @return void
	 */
	public function deleteDeprovisioningNotification(): void {
		$payload["ids"] = ["deprovision"];

		OcsApiHelper::sendRequest(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getAdminUsername(),
			$this->featureContext->getAdminPassword(),
			'DELETE',
			$this->globalNotificationEndpointPath,
			$this->featureContext->getStepLineRef(),
			json_encode($payload)
		);
	}

	/**
	 * @var string
	 */
	private string $userRecipient;

	/**
	 * @param string $userRecipient
	 *
	 * @return void
	 */
	public function setUserRecipient(string $userRecipient): void {
		$this->userRecipient = $userRecipient;
	}

	/**
	 * @return string
	 */
	public function getUserRecipient(): string {
		return $this->userRecipient;
	}

	/**
	 * @BeforeScenario
	 *
	 * @param BeforeScenarioScope $scope
	 *
	 * @return void
	 * @throws Exception
	 */
	public function before(BeforeScenarioScope $scope): void {
		// Get the environment
		$environment = $scope->getEnvironment();
		// Get all the contexts you need in this context
		$this->featureContext = BehatHelper::getContext($scope, $environment, 'FeatureContext');
		$this->spacesContext = BehatHelper::getContext($scope, $environment, 'SpacesContext');
	}

	/**
	 * @param string $user
	 *
	 * @return ResponseInterface
	 */
	public function listAllNotifications(string $user): ResponseInterface {
		$this->setUserRecipient($user);
		$language = SettingsHelper::getLanguageSettingValue(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			$this->featureContext->getStepLineRef()
		);
		$headers = ["accept-language" => $language];
		return OcsApiHelper::sendRequest(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			'GET',
			$this->notificationEndpointPath,
			$this->featureContext->getStepLineRef(),
			[],
			2,
			$headers
		);
	}

	/**
	 * @When /^user "([^"]*)" lists all notifications$/
	 *
	 * @param string $user
	 *
	 * @return void
	 */
	public function userListAllNotifications(string $user): void {
		$response = $this->listAllNotifications($user);
		$this->featureContext->setResponse($response);
		$this->featureContext->pushToLastHttpStatusCodesArray();
	}

	/**
	 * @param string $user
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function deleteAllNotifications(string $user): ResponseInterface {
		$response = $this->listAllNotifications($user);
		if (isset($this->featureContext->getJsonDecodedResponseBodyContent($response)->ocs->data)) {
			$responseBody = $this->featureContext->getJsonDecodedResponseBodyContent($response)->ocs->data;
			foreach ($responseBody as $value) {
				// set notificationId
				$this->notificationIds[] = $value->notification_id;
			}
		}
		return $this->userDeletesNotification($user);
	}

	/**
	 * @When user :user deletes all notifications
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userDeletesAllNotifications(string $user): void {
		$response = $this->deleteAllNotifications($user);
		$this->featureContext->setResponse($response);
	}

	/**
	 * @Given user :user has deleted all notifications
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userHasDeletedAllNotifications(string $user): void {
		$response = $this->deleteAllNotifications($user);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
	}

	/**
	 * @When user :user deletes a notification related to resource :resource with subject :subject
	 *
	 * @param string $user
	 * @param string $resource
	 * @param string $subject
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userDeletesNotificationOfResourceAndSubject(string $user, string $resource, string $subject): void {
		$response = $this->listAllNotifications($user);
		$this->filterNotificationsBySubjectAndResource($subject, $resource, $response);
		$this->featureContext->setResponse($this->userDeletesNotification($user));
	}

	/**
	 * deletes notification
	 *
	 * @param string $user
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userDeletesNotification(string $user): ResponseInterface {
		$this->setUserRecipient($user);
		$payload["ids"] = $this->getNotificationIds();
		return OcsApiHelper::sendRequest(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			'DELETE',
			$this->notificationEndpointPath,
			$this->featureContext->getStepLineRef(),
			\json_encode($payload),
			2
		);
	}

	/**
	 * @Then the notifications should be empty
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theNotificationsShouldBeEmpty(): void {
		$statusCode = $this->featureContext->getResponse()->getStatusCode();
		if ($statusCode !== 200) {
			$response = $this->featureContext->getResponse()->getBody()->getContents();
			throw new \Exception(
				__METHOD__
				. " Failed to get user notification list" . $response
			);
		}
		$notifications = $this->featureContext->getJsonDecodedResponseBodyContent()->ocs->data;
		Assert::assertNull($notifications, "response should not contain any notification");
	}

	/**
	 * @Then user :user should not have any notification
	 *
	 * @param $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldNotHaveAnyNotification($user): void {
		$response = $this->listAllNotifications($user);
		$notifications = $this->featureContext->getJsonDecodedResponseBodyContent($response)->ocs->data;
		Assert::assertNull($notifications, "response should not contain any notification");
	}

	/**
	 * @Then /^there should be "([^"]*)" notifications$/
	 *
	 * @param int $numberOfNotification
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldHaveNotifications(int $numberOfNotification): void {
		if (!isset($this->featureContext->getJsonDecodedResponseBodyContent()->ocs->data)) {
			throw new Exception("Notification is empty");
		}
		$responseBody = $this->featureContext->getJsonDecodedResponseBodyContent()->ocs->data;
		$actualNumber = \count($responseBody);
		Assert::assertEquals(
			$numberOfNotification,
			$actualNumber,
			"Expected number of notifications was '$numberOfNotification', but got '$actualNumber'"
		);
	}

	/**
	 * @Then /^the JSON response should contain a notification message with the subject "([^"]*)" and the message-details should match$/
	 *
	 * @param string $subject
	 * @param PyStringNode $schemaString
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theJsonDataFromLastResponseShouldMatch(
		string $subject,
		PyStringNode $schemaString
	): void {
		$responseBody = $this->filterResponseAccordingToNotificationSubject($subject);
		// substitute the value here
		$schemaString = $schemaString->getRaw();
		$schemaString = $this->featureContext->substituteInLineCodes(
			$schemaString,
			$this->featureContext->getCurrentUser(),
			[],
			[],
			null,
			$this->getUserRecipient(),
		);
		$this->featureContext->assertJsonDocumentMatchesSchema(
			$responseBody,
			$this->featureContext->getJSONSchema($schemaString)
		);
	}

	/**
	 * filter notification according to subject
	 *
	 * @param string $subject
	 * @param ResponseInterface|null $response
	 *
	 * @return object
	 */
	public function filterResponseAccordingToNotificationSubject(
		string $subject,
		?ResponseInterface $response = null
	): object {
		$response = $response ?? $this->featureContext->getResponse();
		if (isset($this->featureContext->getJsonDecodedResponseBodyContent($response)->ocs->data)) {
			$responseBody = $this->featureContext->getJsonDecodedResponseBodyContent($response)->ocs->data;
			foreach ($responseBody as $value) {
				if (isset($value->subject) && $value->subject === $subject) {
					$responseBody = $value;
					// set notificationId
					$this->notificationIds[] = $value->notification_id;
					break;
				}
			}
		} else {
			$responseBody = $this->featureContext->getJsonDecodedResponseBodyContent($response);
		}
		return $responseBody;
	}

	/**
	 * filter notification according to subject and resource
	 *
	 * @param string $subject
	 * @param string $resource
	 * @param ResponseInterface|null $response
	 *
	 * @return array
	 */
	public function filterNotificationsBySubjectAndResource(
		string $subject,
		string $resource,
		?ResponseInterface $response = null
	): array {
		$filteredNotifications = [];
		$response = $response ?? $this->featureContext->getResponse();
		$responseObject = $this->featureContext->getJsonDecodedResponseBodyContent($response);

		if (!isset($responseObject->ocs->data)) {
			Assert::fail("Response doesn't contain notification: " . print_r($responseObject, true));
		}

		$notifications = $responseObject->ocs->data;
		foreach ($notifications as $notification) {
			if (isset($notification->subject) && $notification->subject === $subject
				&& isset($notification->messageRichParameters->resource->name)
				&& $notification->messageRichParameters->resource->name === $resource
			) {
				$this->notificationIds[] = $notification->notification_id;
				$filteredNotifications[] = $notification;
			}
		}
		return $filteredNotifications;
	}

	/**
	 * @Then /^user "([^"]*)" should (?:get|have) a notification with subject "([^"]*)" and message:$/
	 *
	 * @param string $user
	 * @param string $subject
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldGetANotificationWithMessage(string $user, string $subject, TableNode $table): void {
		$count = 0;
		// Sometimes the test might try to get the notifications before the server has created the notification.
		// To prevent the test from failing because of that, try to list the notifications again
		do {
			if ($count > 0) {
				\sleep(1);
			}
			$this->featureContext->setResponse(null);
			$response = $this->listAllNotifications($user);
			$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
			++$count;
		} while (!isset($this->filterResponseAccordingToNotificationSubject($subject, $response)->message)
			&& $count <= 10
		);
		if (isset($this->filterResponseAccordingToNotificationSubject($subject, $response)->message)) {
			$actualMessage = str_replace(
				["\r", "\n"],
				" ",
				$this->filterResponseAccordingToNotificationSubject($subject, $response)->message
			);
		} else {
			throw new \Exception("Notification was not found even after retrying for 5 seconds.");
		}
		$expectedMessage = $table->getColumnsHash()[0]['message'];
		Assert::assertSame(
			$expectedMessage,
			$actualMessage,
			__METHOD__ . "expected message to be '$expectedMessage' but found'$actualMessage'"
		);
	}

	/**
	 * @Then user :user should get a notification for resource :resource with subject :subject and message:
	 *
	 * @param string $user
	 * @param string $resource
	 * @param string $subject
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldGetNotificationForResourceWithMessage(
		string $user,
		string $resource,
		string $subject,
		TableNode $table
	): void {
		$response = $this->listAllNotifications($user);
		$notification = $this->filterNotificationsBySubjectAndResource($subject, $resource, $response);

		if (\count($notification) === 1) {
			$actualMessage = str_replace(["\r", "\r"], " ", $notification[0]->message);
			$expectedMessage = $table->getColumnsHash()[0]['message'];
			Assert::assertSame(
				$expectedMessage,
				$actualMessage,
				__METHOD__ . "expected message to be '$expectedMessage' but found'$actualMessage'"
			);
			$response = $this->userDeletesNotification($user);
			$this->featureContext->theHTTPStatusCodeShouldBe(200, '', $response);
		} elseif (\count($notification) === 0) {
			throw new \Exception(
				"Response doesn't contain any notification with resource '$resource' and subject '$subject'.\n"
				. print_r($notification, true)
			);
		} else {
			throw new \Exception(
				"Response contains more than one notification with resource '$resource' and subject '$subject'.\n"
				. print_r($notification, true)
			);
		}
	}

	/**
	 * @Then user :user should not have a notification related to resource :resource with subject :subject
	 *
	 * @param string $user
	 * @param string $resource
	 * @param string $subject
	 *
	 * @return void
	 */
	public function userShouldNotHaveANotificationRelatedToResourceWithSubject(
		string $user,
		string $resource,
		string $subject
	): void {
		$response = $this->listAllNotifications($user);
		$filteredResponse = $this->filterNotificationsBySubjectAndResource($subject, $resource, $response);
		Assert::assertCount(
			0,
			$filteredResponse,
			"Response should not contain notification related to resource '$resource' with subject '$subject' but found"
			. print_r($filteredResponse, true)
		);
	}

	/**
	 * @Then user :user should have received the following email from user :sender about the share of project space :spaceName
	 *
	 * @param string $user
	 * @param string $sender
	 * @param string $spaceName
	 * @param PyStringNode $content
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldHaveReceivedTheFollowingEmailFromUserAboutTheShareOfProjectSpace(
		string $user,
		string $sender,
		string $spaceName,
		PyStringNode $content
	): void {
		$rawExpectedEmailBodyContent = \str_replace("\r\n", "\n", $content->getRaw());
		$this->featureContext->setResponse(
			GraphHelper::getMySpaces(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				'',
				$this->featureContext->getStepLineRef()
			)
		);
		$expectedEmailBodyContent = $this->featureContext->substituteInLineCodes(
			$rawExpectedEmailBodyContent,
			$sender,
			[],
			[
				[
					"code" => "%space_id%",
					"function" =>
						[$this->spacesContext, "getSpaceIdByName"],
					"parameter" => [$sender, $spaceName]
				],
			]
		);
		$this->assertEmailContains($user, $expectedEmailBodyContent);
	}

	/**
	 * @Then user :user should have received the following email from user :sender
	 *
	 * @param string $user
	 * @param string $sender
	 * @param PyStringNode $content
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldHaveReceivedTheFollowingEmailFromUser(
		string $user,
		string $sender,
		PyStringNode $content
	): void {
		$rawExpectedEmailBodyContent = \str_replace("\r\n", "\n", $content->getRaw());
		$expectedEmailBodyContent = $this->featureContext->substituteInLineCodes(
			$rawExpectedEmailBodyContent,
			$sender
		);
		$this->assertEmailContains($user, $expectedEmailBodyContent);
	}

	/**
	 * @Then user :user should have received the following email from user :sender ignoring whitespaces
	 *
	 * @param string $user
	 * @param string $sender
	 * @param PyStringNode $content
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldHaveReceivedTheFollowingEmailFromUserIgnoringWhitespaces(
		string $user,
		string $sender,
		PyStringNode $content
	): void {
		$rawExpectedEmailBodyContent = \str_replace("\r\n", "\n", $content->getRaw());
		$expectedEmailBodyContent = $this->featureContext->substituteInLineCodes(
			$rawExpectedEmailBodyContent,
			$sender
		);
		$this->assertEmailContains($user, $expectedEmailBodyContent, true);
	}

	/***
	 * @param string $user
	 * @param string $expectedEmailBodyContent
	 * @param bool $ignoreWhiteSpace
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function assertEmailContains(
		string $user,
		string $expectedEmailBodyContent,
		$ignoreWhiteSpace = false
	): void {
		$address = $this->featureContext->getEmailAddressForUser($user);
		$this->featureContext->pushEmailRecipientAsMailBox($address);

		// assert with retries as email delivery might be delayed
		$retried = 0;
		do {
			$actualEmailBodyContent = EmailHelper::getBodyOfLastEmail(
				$address,
				$this->featureContext->getStepLineRef()
			);
			if ($ignoreWhiteSpace) {
				$expectedEmailBodyContent = preg_replace('/\s+/', '', $expectedEmailBodyContent);
				$actualEmailBodyContent = preg_replace('/\s+/', '', $actualEmailBodyContent);
			}
			$tryAgain = !\str_contains($actualEmailBodyContent, $expectedEmailBodyContent)
				&& $retried <= STANDARD_RETRY_COUNT;
			$retried++;
			if ($tryAgain) {
				$mailBox = EmailHelper::getMailBoxFromEmail($address);
				echo "[INFO] Checking last email content for '$mailBox'. (Retry $retried)\n";
				// wait for 1 second before trying again
				sleep(1);
			}
		} while ($tryAgain);
		Assert::assertStringContainsString(
			$expectedEmailBodyContent,
			$actualEmailBodyContent,
			"The email address '$address' should have received an"
			. " email with the body containing '$expectedEmailBodyContent'"
			. " but the received email is '$actualEmailBodyContent'"
		);
	}

	/**
	 * Delete all emails from the mailboxes
	 *
	 * @AfterScenario @email
	 *
	 * @return void
	 */
	public function clearMailboxes(): void {
		$users = \array_keys($this->featureContext->getCreatedUsers());
		try {
			if (!empty($users)) {
				foreach ($users as $emailRecipient) {
					$retried = 0;
					do {
						$res = EmailHelper::deleteAllEmails(
							EmailHelper::getLocalEmailUrl(),
							$emailRecipient,
							$this->featureContext->getStepLineRef(),
						);
						$deleteStatus = $res->getStatusCode();
						$mailBox = EmailHelper::getMailboxInformation($emailRecipient);
						$tryAgain = ($deleteStatus !== 200 || !empty($mailBox)) && $retried <= STANDARD_RETRY_COUNT;
						$retried++;
						if ($tryAgain) {
							echo "[INFO] Clearing mailbox '$emailRecipient'."
							. " Status: $deleteStatus. Emails: " . \count($mailBox) . "."
							. " (Retry $retried)\n";
							// wait for 1 second before trying again
							sleep(1);
						}
					} while ($tryAgain);
				}
			}
		} catch (Exception $e) {
			echo __METHOD__ .
				" could not delete inbucket messages, is inbucket set up?\n" .
				$e->getMessage();
		}
	}

	/**
	 *
	 * @param string|null $user
	 * @param string|null $deprovision_date
	 * @param string|null $deprovision_date_format
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 *
	 * @throws JsonException
	 */
	public function userCreatesDeprovisioningNotification(
		?string $user = null,
		?string $deprovision_date = "2043-07-04T11:23:12Z",
		?string $deprovision_date_format= "2006-01-02T15:04:05Z07:00"
	): ResponseInterface {
		$payload["type"] = "deprovision";
		$payload["data"] = [
			"deprovision_date" => $deprovision_date, "deprovision_date_format" => $deprovision_date_format];
		return OcsApiHelper::sendRequest(
			$this->featureContext->getBaseUrl(),
			$user ? $this->featureContext->getActualUsername($user) : $this->featureContext->getAdminUsername(),
			$user ? $this->featureContext->getPasswordForUser($user) : $this->featureContext->getAdminPassword(),
			'POST',
			$this->globalNotificationEndpointPath,
			$this->featureContext->getStepLineRef(),
			json_encode($payload)
		);
	}

	/**
	 * @When the administrator creates a deprovisioning notification
	 * @When user :user tries to create a deprovisioning notification
	 *
	 * @param string|null $user
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function theAdministratorCreatesADeprovisioningNotification(?string $user = null) {
		$response = $this->userCreatesDeprovisioningNotification($user);
		$this->featureContext->setResponse($response);
		$this->featureContext->pushToLastHttpStatusCodesArray();
	}

	/**
	 * @When the administrator creates a deprovisioning notification for date :deprovision_date of format :deprovision_date_format
	 *
	 * @param $deprovision_date
	 * @param $deprovision_date_format
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function theAdministratorCreatesADeprovisioningNotificationUsingDateFormat(
		$deprovision_date,
		$deprovision_date_format
	) {
		$response = $this->userCreatesDeprovisioningNotification(null, $deprovision_date, $deprovision_date_format);
		$this->featureContext->setResponse($response);
		$this->featureContext->pushToLastHttpStatusCodesArray();
	}

	/**
	 * @Given the administrator has created a deprovisioning notification
	 *
	 * @return void
	 */
	public function userHasCreatedDeprovisioningNotification(): void {
		$response = $this->userCreatesDeprovisioningNotification();
		$this->featureContext->theHTTPStatusCodeShouldBe(200, "", $response);
	}

	/**
	 * @When the administrator deletes the deprovisioning notification
	 * @When user :user tries to delete the deprovisioning notification
	 *
	 * @param string|null $user
	 *
	 * @return void
	 */
	public function userDeletesDeprovisioningNotification(?string $user = null): void {
		$payload["ids"] = ["deprovision"];

		$response = OcsApiHelper::sendRequest(
			$this->featureContext->getBaseUrl(),
			$user ? $this->featureContext->getActualUsername($user) : $this->featureContext->getAdminUsername(),
			$user ? $this->featureContext->getPasswordForUser($user) : $this->featureContext->getAdminPassword(),
			'DELETE',
			$this->globalNotificationEndpointPath,
			$this->featureContext->getStepLineRef(),
			json_encode($payload)
		);
		$this->featureContext->setResponse($response);
	}
}
