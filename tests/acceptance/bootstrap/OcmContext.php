<?php declare(strict_types=1);
/**
 * @author Viktor Scharf <scharf.vi@gmail.com>
 *
 * @copyright Copyright (c) 2024, ownCloud GmbH
 * @license AGPL-3.0
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

use Behat\Behat\Context\Context;
use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use GuzzleHttp\Exception\GuzzleException;
use Psr\Http\Message\ResponseInterface;
use TestHelpers\OcHelper;
use TestHelpers\OcmHelper;
use TestHelpers\WebDavHelper;
use TestHelpers\BehatHelper;
use Behat\Step\Given;
use Behat\Step\When;

/**
 * Acceptance test steps related to testing federation share(ocm) features
 */
class OcmContext implements Context {
	private FeatureContext $featureContext;
	private string $invitationToken;

	/**
	 * This will run before EVERY scenario.
	 * It will set the properties for this object.
	 *
	 * @BeforeScenario
	 *
	 * @param BeforeScenarioScope $scope
	 *
	 * @return void
	 */
	public function before(BeforeScenarioScope $scope): void {
		// Get the environment
		$environment = $scope->getEnvironment();
		// Get all the contexts you need in this context from here
		$this->featureContext = BehatHelper::getContext($scope, $environment, 'FeatureContext');
	}

	/**
	 * @return string
	 * @throws Exception
	 */
	public function getLastFederatedInvitationToken(): string {
		if (empty($this->invitationToken)) {
			throw new Exception(__METHOD__ . " token not found");
		}
		return $this->invitationToken;
	}

	/**
	 * @param string $user
	 * @param string $email
	 * @param string $description
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function createInvitation(string $user, $email = null, $description = null): ResponseInterface {
		$response = OcmHelper::createInvitation(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$email,
			$description
		);
		$responseData = \json_decode($response->getBody()->getContents(), true, 512, JSON_THROW_ON_ERROR);
		if (isset($responseData["token"])) {
			$this->invitationToken = $responseData["token"];
		}
		return $response;
	}

	/**
	 *
	 * @param string $user
	 * @param string $email
	 * @param string $description
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When(':user creates the federation share invitation')]
	#[When(':user creates the federation share invitation with email :email and description :description')]
	public function userCreatesTheFederationShareInvitation(string $user, $email = null, $description = null): void {
		$this->featureContext->setResponse($this->createInvitation($user, $email, $description));
	}

	/**
	 *
	 * @param string $user
	 * @param string $email
	 * @param string $description
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Given(':user has created the federation share invitation')]
	#[Given(':user has created the federation share invitation with email :email and description :description')]
	public function userHasCreatedTheFederationShareInvitation(string $user, $email = null, $description = null): void {
		$response = $this->createInvitation($user, $email, $description);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, '', $response);
	}

	/**
	 * @param string $user
	 * @param string $token
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function acceptInvitation(string $user, ?string $token = null): ResponseInterface {
		$providerDomain = $this->featureContext->getLocalBaseUrlWithoutScheme();
		if ($this->featureContext->getCurrentServer() === "LOCAL") {
			$providerDomain = $this->featureContext->getRemoteBaseUrlWithoutScheme();
		}
		return OcmHelper::acceptInvitation(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$token ? $token : $this->getLastFederatedInvitationToken(),
			$providerDomain
		);
	}

	/**
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When(':user accepts the last federation share invitation')]
	#[When(':user tries to accept the last federation share invitation')]
	public function userAcceptsTheLastFederationShareInvitation(string $user): void {
		$this->featureContext->setResponse($this->acceptInvitation($user));
	}

	/**
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When(':user tries to accept the invitation with invalid token')]
	public function userTriesToAcceptInvitationWithInvalidToken(string $user): void {
		$this->featureContext->setResponse($this->acceptInvitation($user, WebDavHelper::generateUUIDv4()));
	}

	/**
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[Given(':user has accepted invitation')]
	public function userHasAcceptedTheLastFederationShareInvitation(string $user): void {
		$response = $this->acceptInvitation($user);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, '', $response);
	}

	/**
	 * @param string $user
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function findAcceptedUsers(string $user): ResponseInterface {
		return OcmHelper::findAcceptedUsers(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user)
		);
	}

	/**
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When(':user searches for accepted users')]
	public function userFindsAcceptedUsers(string $user): void {
		$this->featureContext->setResponse($this->findAcceptedUsers($user));
	}

	/**
	 *
	 * @param string $user
	 * @param string $ocmUserName
	 *
	 * @return array
	 * @throws GuzzleException
	 */
	public function getAcceptedUserByName(string $user, string $ocmUserName): array {
		$users = ($this->featureContext->getJsonDecodedResponse($this->findAcceptedUsers($user)));
		foreach ($users as $user) {
			if (strpos($user["display_name"], $ocmUserName) !== false) {
				return $user;
			}
		}
		throw new \Exception("Could not find user with name '{$ocmUserName}' in the accepted users list.");
	}

	/**
	 * @param string $user
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function listInvitations(string $user): ResponseInterface {
		return OcmHelper::listInvite(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user)
		);
	}

	/**
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When(':user lists the created invitations')]
	public function userListsCreatedInvitations(string $user): void {
		$this->featureContext->setResponse($this->listInvitations($user));
	}

	/**
	 *
	 * @param int $number
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('the user waits :number seconds for the invitation token to expire')]
	public function theUserWaitsForTokenToExpire(int $number): void {
		\sleep($number);
	}

	/**
	 *
	 * @param string $user
	 * @param string $ocmUser
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('user :user deletes federated connection with user :ocmUser using the Graph API')]
	public function userDeletesFederatedConnectionWithUserUsingTheGraphApi(string $user, string $ocmUser): void {
		$this->featureContext->setResponse($this->deleteConnection($user, $ocmUser));
	}

	/**
	 *
	 * @param string $user
	 * @param string $ocmUser
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	#[When('user :user has deleted federated connection with user :ocmUser')]
	public function userHasDeletedFederatedConnectionWithUser(string $user, string $ocmUser): void {
		$response = $this->deleteConnection($user, $ocmUser);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"failed while deleting connection with user $ocmUser",
			$response
		);
	}

	/**
	 * @param string $user
	 * @param string $ocmUser
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function deleteConnection(string $user, string $ocmUser): ResponseInterface {
		$ocmUser = $this->getAcceptedUserByName($user, $ocmUser);
		return OcmHelper::deleteConnection(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$ocmUser['user_id'],
			$ocmUser['idp']
		);
	}
}
