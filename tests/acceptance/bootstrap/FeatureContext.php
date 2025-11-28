<?php declare(strict_types=1);

/**
 * @author Sergio Bertolin <sbertolin@owncloud.com>
 * @author Phillip Davis <phil@jankaritech.com>
 * @copyright Copyright (c) 2018, ownCloud GmbH
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

use Behat\Behat\Hook\Scope\BeforeStepScope;
use GuzzleHttp\Exception\GuzzleException;
use rdx\behatvars\BehatVariablesContext;
use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use Behat\Behat\Hook\Scope\AfterScenarioScope;
use Behat\Gherkin\Node\PyStringNode;
use Behat\Gherkin\Node\TableNode;
use Behat\Testwork\Hook\Scope\BeforeSuiteScope;
use GuzzleHttp\Cookie\CookieJar;
use Psr\Http\Message\ResponseInterface;
use PHPUnit\Framework\Assert;
use Swaggest\JsonSchema\Schema as JsonSchema;
use Laminas\Ldap\Ldap;
use TestHelpers\SetupHelper;
use TestHelpers\HttpRequestHelper;
use TestHelpers\HttpLogger;
use TestHelpers\OcHelper;
use TestHelpers\GraphHelper;
use TestHelpers\WebDavHelper;
use TestHelpers\SettingsHelper;
use TestHelpers\OcConfigHelper;
use TestHelpers\BehatHelper;
use Swaggest\JsonSchema\InvalidValue as JsonSchemaException;
use Swaggest\JsonSchema\Exception\ArrayException;
use Swaggest\JsonSchema\Exception\ConstException;
use Swaggest\JsonSchema\Exception\ContentException;
use Swaggest\JsonSchema\Exception\EnumException;
use Swaggest\JsonSchema\Exception\LogicException;
use Swaggest\JsonSchema\Exception\NumericException;
use Swaggest\JsonSchema\Exception\ObjectException;
use Swaggest\JsonSchema\Exception\StringException;
use Swaggest\JsonSchema\Exception\TypeException;

require_once 'bootstrap.php';

/**
 * Features context.
 */
class FeatureContext extends BehatVariablesContext {
	use Provisioning;
	use Sharing;
	use WebDav;

	/**
	 * json schema validator keywords
	 * See: https://json-schema.org/draft-06/draft-wright-json-schema-validation-01#rfc.section.6
	 */
	private array $jsonSchemaValidators;

	/**
	 * Unix timestamp seconds
	 */
	private int $scenarioStartTime;
	private string $adminUsername;
	private string $adminPassword;
	private string $originalAdminPassword;

	/**
	 * An array of values of replacement values of user attributes.
	 * These are only referenced when creating a user. After that, the
	 * run-time values are maintained and referenced in the $createdUsers array.
	 *
	 * Key is the username, value is an array of user attributes
	 */
	private ?array $userReplacements = null;
	private string $regularUserPassword;
	private string $alt1UserPassword;
	private string $alt2UserPassword;
	private string $alt3UserPassword;
	private string $alt4UserPassword;

	/**
	 * The password to use in tests that create a sub-admin user
	 */
	private string $subAdminPassword;

	/**
	 * The password to use in tests that create another admin user
	 */
	private string $alternateAdminPassword;

	/**
	 * The password to use in tests that create public link shares
	 */
	private string $publicLinkSharePassword;
	private string $currentUser = '';
	private string $currentServer;

	/**
	 * The base URL of the current server under test,
	 * without any terminating slash
	 * e.g. http://localhost:8080
	 */
	private string $baseUrl;

	/**
	 * The base URL of the local server under test,
	 * without any terminating slash
	 * e.g. http://localhost:8080
	 */
	private string $localBaseUrl;

	/**
	 * The base URL of the remote (federated) server under test,
	 * without any terminating slash
	 * e.g. http://localhost:8180
	 */
	private string $remoteBaseUrl;

	/**
	 * The suite name, feature name and scenario line number.
	 * Example: apiComments/createComments.feature:24
	 */
	private string $scenarioString = '';

	/**
	 * A full unique reference to the step that is currently executing.
	 * Example: apiComments/createComments.feature:24-28
	 * That is line 28, in the scenario at line 24, in the createComments feature
	 * in the apiComments suite.
	 */
	private string $stepLineRef = '';

	private int $ocsApiVersion = 1;
	private ?ResponseInterface $response = null;
	private string $responseUser = '';
	public array $emailRecipients = [];
	private CookieJar $cookieJar;
	private string $requestToken;
	private array $createdFiles = [];

	/**
	 * The local source IP address from which to initiate API actions.
	 * Defaults to system-selected address matching IP address family and scope.
	 */
	private ?string $sourceIpAddress = null;
	private array $guzzleClientHeaders = [];
	public OCSContext $ocsContext;
	public AuthContext $authContext;
	public TUSContext $tusContext;
	public GraphContext $graphContext;
	public SpacesContext $spacesContext;
	public OcmContext $ocmContext;

	/**
	 * The codes are stored as strings, even though they are numbers
	 */
	private array $lastHttpStatusCodesArray = [];
	private array $lastOCSStatusCodesArray = [];

	/**
	 * Store for auto-sync settings for users
	 */
	private array $autoSyncSettings = [];

	/**
	 * @param string $user
	 *
	 * @return bool
	 */
	public function getUserAutoSyncSetting(string $user): bool {
		if (\array_key_exists($user, $this->autoSyncSettings)) {
			return $this->autoSyncSettings[$user];
		}
		$autoSyncSetting = SettingsHelper::getAutoAcceptSharesSettingValue(
			$this->baseUrl,
			$user,
			$this->getPasswordForUser($user),
			$this->getStepLineRef()
		);
		$this->autoSyncSettings[$user] = $autoSyncSetting;

		return $autoSyncSetting;
	}

	/**
	 * @param string $user
	 * @param bool $value
	 *
	 * @return void
	 */
	public function rememberUserAutoSyncSetting(string $user, bool $value): void {
		$this->autoSyncSettings[$user] = $value;
	}

	private bool $useSharingNG = false;

	/**
	 * @return bool
	 */
	public function isUsingSharingNG(): bool {
		return $this->useSharingNG;
	}

	/**
	 * @param string|null $httpStatusCode
	 *
	 * @return void
	 */
	public function pushToLastHttpStatusCodesArray(?string $httpStatusCode = null): void {
		if ($httpStatusCode !== null) {
			$this->lastHttpStatusCodesArray[] = $httpStatusCode;
		} elseif ($this->getResponse()->getStatusCode() !== null) {
			$this->lastHttpStatusCodesArray[] = (string)$this->getResponse()->getStatusCode();
		}
	}

	/**
	 * @return void
	 */
	public function emptyLastHTTPStatusCodesArray(): void {
		$this->lastHttpStatusCodesArray = [];
	}

	/**
	 * @return void
	 */
	public function emptyLastOCSStatusCodesArray(): void {
		$this->lastOCSStatusCodesArray = [];
	}

	/**
	 * @return void
	 */
	public function clearStatusCodeArrays(): void {
		$this->emptyLastHTTPStatusCodesArray();
		$this->emptyLastOCSStatusCodesArray();
	}

	/**
	 * @param string $ocsStatusCode
	 *
	 * @return void
	 */
	public function pushToLastOcsCodesArray(string $ocsStatusCode): void {
		$this->lastOCSStatusCodesArray[] = $ocsStatusCode;
	}

	/**
	 * Add HTTP and OCS status code of the last response to the respective status code array
	 *
	 * @return void
	 */
	public function pushToLastStatusCodesArrays(): void {
		$this->pushToLastHttpStatusCodesArray(
			(string)$this->getResponse()->getStatusCode()
		);
		try {
			$this->pushToLastOcsCodesArray(
				$this->ocsContext->getOCSResponseStatusCode(
					$this->getResponse()
				)
			);
		} catch (Exception $exception) {
			// if response couldn't be converted into xml then push "notset" to last ocs status codes array
			$this->pushToLastOcsCodesArray("notset");
		}
	}

	/**
	 * @param string $emailAddress
	 *
	 * @return void
	 */
	public function pushEmailRecipientAsMailBox(string $emailAddress): void {
		$mailBox = explode("@", $emailAddress)[0];
		if (!\in_array($mailBox, $this->emailRecipients)) {
			$this->emailRecipients[] = $mailBox;
		}
	}

	private Ldap $ldap;
	private string $ldapBaseDN;
	private string $ldapHost;
	private int $ldapPort;
	private string $ldapAdminUser;
	private string $ldapAdminPassword = "";
	private string $ldapUsersOU;
	private string $ldapGroupsOU;
	private string $ldapGroupSchema;
	private bool $skipImportLdif;
	private array $toDeleteDNs = [];
	private array $ldapCreatedUsers = [];
	private array $ldapCreatedGroups = [];
	private array $toDeleteLdapConfigs = [];
	private array $oldLdapConfig = [];

	/**
	 * @return Ldap
	 */
	public function getLdap(): Ldap {
		return $this->ldap;
	}

	/**
	 * @param string $configId
	 *
	 * @return void
	 */
	public function setToDeleteLdapConfigs(string $configId): void {
		$this->toDeleteLdapConfigs[] = $configId;
	}

	/**
	 * @return array
	 */
	public function getToDeleteLdapConfigs(): array {
		return $this->toDeleteLdapConfigs;
	}

	/**
	 * @param string $setValue
	 *
	 * @return void
	 */
	public function setToDeleteDNs(string $setValue): void {
		$this->toDeleteDNs[] = $setValue;
	}

	/**
	 * @return string
	 */
	public function getLdapBaseDN(): string {
		return $this->ldapBaseDN;
	}

	/**
	 * @return string
	 */
	public function getLdapUsersOU(): string {
		return $this->ldapUsersOU;
	}

	/**
	 * @return string
	 */
	public function getLdapGroupsOU(): string {
		return $this->ldapGroupsOU;
	}

	/**
	 * @return array
	 */
	public function getOldLdapConfig(): array {
		return $this->oldLdapConfig;
	}

	/**
	 * @param string $configId
	 * @param string $configKey
	 * @param string $value
	 *
	 * @return void
	 */
	public function setOldLdapConfig(string $configId, string $configKey, string $value): void {
		$this->oldLdapConfig[$configId][$configKey] = $value;
	}

	/**
	 * @return string
	 */
	public function getLdapHost(): string {
		return $this->ldapHost;
	}

	/**
	 * @return string
	 */
	public function getLdapHostWithoutScheme(): string {
		return $this->removeSchemeFromUrl($this->ldapHost);
	}

	/**
	 * @return integer
	 */
	public function getLdapPort(): int {
		return $this->ldapPort;
	}

	/**
	 * @return bool
	 */
	public function isTestingWithLdap(): bool {
		return (\getenv("TEST_WITH_LDAP") === "true");
	}

	/**
	 * @return bool
	 */
	public function isTestingReplacingUsernames(): bool {
		return (\getenv('REPLACE_USERNAMES') === "true");
	}

	/**
	 * @return array|null
	 */
	public function usersToBeReplaced(): ?array {
		if (($this->userReplacements === null) && $this->isTestingReplacingUsernames()) {
			$this->userReplacements = \json_decode(
				\file_get_contents("./tests/acceptance/usernames.json"),
				true
			);
			// Loop through the user replacements, and make entries for the lower
			// and upper case forms. This allows for steps that specifically
			// want to test that usernames like "alice", "Alice" and "ALICE" all work.
			// Such steps will make useful replacements for each form.
			foreach ($this->userReplacements as $key => $value) {
				$lowerKey = \strtolower($key);
				if ($lowerKey !== $key) {
					$this->userReplacements[$lowerKey] = $value;
					$this->userReplacements[$lowerKey]['username'] = \strtolower(
						$this->userReplacements[$lowerKey]['username']
					);
				}
				$upperKey = \strtoupper($key);
				if ($upperKey !== $key) {
					$this->userReplacements[$upperKey] = $value;
					$this->userReplacements[$upperKey]['username'] = \strtoupper(
						$this->userReplacements[$upperKey]['username']
					);
				}
			}
		}
		return $this->userReplacements;
	}

	/**
	 * BasicStructure constructor.
	 *
	 * @param string $adminUsername
	 * @param string $adminPassword
	 * @param string $regularUserPassword
	 *
	 */
	public function __construct(
		string $adminUsername,
		string $adminPassword,
		string $regularUserPassword,
	) {
		// Initialize your context here
		$this->adminUsername = $adminUsername;
		$this->adminPassword = $adminPassword;
		$this->regularUserPassword = $regularUserPassword;
		$this->currentServer = 'LOCAL';
		$this->cookieJar = new CookieJar();

		// These passwords are referenced in tests and can be overridden by
		// setting environment variables.
		$this->alt1UserPassword = "1234";
		$this->alt2UserPassword = "AaBb2Cc3Dd4";
		$this->alt3UserPassword = "aVeryLongPassword42TheMeaningOfLife";
		$this->alt4UserPassword = "ThisIsThe4thAlternatePwd";
		$this->subAdminPassword = "IamAJuniorAdmin42";
		$this->alternateAdminPassword = "IHave99LotsOfPriv";
		$this->publicLinkSharePassword = "publicPwd:1";

		$this->baseUrl = OcHelper::getServerUrl();
		$this->localBaseUrl = $this->baseUrl;
		// federated server url from the environment
		$this->remoteBaseUrl = OcHelper::getFederatedServerUrl();

		// get the admin username from the environment (if defined)
		$adminUsernameFromEnvironment = $this->getAdminUsernameFromEnvironment();
		if ($adminUsernameFromEnvironment !== false) {
			$this->adminUsername = $adminUsernameFromEnvironment;
		}

		// get the admin password from the environment (if defined)
		$adminPasswordFromEnvironment = $this->getAdminPasswordFromEnvironment();
		if ($adminPasswordFromEnvironment !== false) {
			$this->adminPassword = $adminPasswordFromEnvironment;
		}

		// get the regular user password from the environment (if defined)
		$regularUserPasswordFromEnvironment = $this->getRegularUserPasswordFromEnvironment();
		if ($regularUserPasswordFromEnvironment !== false) {
			$this->regularUserPassword = $regularUserPasswordFromEnvironment;
		}

		// get the alternate(1) user password from the environment (if defined)
		$alt1UserPasswordFromEnvironment = $this->getAlt1UserPasswordFromEnvironment();
		if ($alt1UserPasswordFromEnvironment !== false) {
			$this->alt1UserPassword = $alt1UserPasswordFromEnvironment;
		}

		// get the alternate(2) user password from the environment (if defined)
		$alt2UserPasswordFromEnvironment = $this->getAlt2UserPasswordFromEnvironment();
		if ($alt2UserPasswordFromEnvironment !== false) {
			$this->alt2UserPassword = $alt2UserPasswordFromEnvironment;
		}

		// get the alternate(3) user password from the environment (if defined)
		$alt3UserPasswordFromEnvironment = $this->getAlt3UserPasswordFromEnvironment();
		if ($alt3UserPasswordFromEnvironment !== false) {
			$this->alt3UserPassword = $alt3UserPasswordFromEnvironment;
		}

		// get the alternate(4) user password from the environment (if defined)
		$alt4UserPasswordFromEnvironment = $this->getAlt4UserPasswordFromEnvironment();
		if ($alt4UserPasswordFromEnvironment !== false) {
			$this->alt4UserPassword = $alt4UserPasswordFromEnvironment;
		}

		// get the sub-admin password from the environment (if defined)
		$subAdminPasswordFromEnvironment = $this->getSubAdminPasswordFromEnvironment();
		if ($subAdminPasswordFromEnvironment !== false) {
			$this->subAdminPassword = $subAdminPasswordFromEnvironment;
		}

		// get the alternate admin password from the environment (if defined)
		$alternateAdminPasswordFromEnvironment = $this->getAlternateAdminPasswordFromEnvironment();
		if ($alternateAdminPasswordFromEnvironment !== false) {
			$this->alternateAdminPassword = $alternateAdminPasswordFromEnvironment;
		}

		// get the public link share password from the environment (if defined)
		$publicLinkSharePasswordFromEnvironment = $this->getPublicLinkSharePasswordFromEnvironment();
		if ($publicLinkSharePasswordFromEnvironment !== false) {
			$this->publicLinkSharePassword = $publicLinkSharePasswordFromEnvironment;
		}
		$this->originalAdminPassword = $this->adminPassword;

		$this->jsonSchemaValidators = \array_keys(JsonSchema::properties()->getDataKeyMap());
	}

	/**
	 * Create log directory if it doesn't exist
	 *
	 * @BeforeSuite
	 *
	 * @param BeforeSuiteScope $scope
	 *
	 * @return void
	 * @throws Exception
	 */
	public static function setupLogDir(BeforeSuiteScope $scope): void {
		if (!\file_exists(HttpLogger::getLogDir())) {
			\mkdir(HttpLogger::getLogDir(), 0777, true);
		}
	}

	/**
	 *
	 * @BeforeScenario
	 *
	 * @param BeforeScenarioScope $scope
	 *
	 * @return void
	 * @throws Exception
	 */
	public static function logScenario(BeforeScenarioScope $scope): void {
		$scenarioLine = self::getScenarioLine($scope);

		if ($scope->getScenario()->getNodeType() === "Example") {
			$scenario = "Scenario Outline: " . $scope->getScenario()->getOutlineTitle();
		} else {
			$scenario = $scope->getScenario()->getNodeType() . ": " . $scope->getScenario()->getTitle();
		}

		$logMessage = "## $scenario ($scenarioLine)\n";

		// Delete previous scenario's log file
		if (\file_exists(HttpLogger::getScenarioLogPath())) {
			\unlink(HttpLogger::getScenarioLogPath());
		}

		// Write the scenario log
		HttpLogger::writeLog(HttpLogger::getScenarioLogPath(), $logMessage);
	}

	/**
	 *
	 * @BeforeStep
	 *
	 * @param BeforeStepScope $scope
	 *
	 * @return void
	 * @throws Exception
	 */
	public static function logStep(BeforeStepScope $scope): void {
		$step = $scope->getStep()->getKeyword() . " " . $scope->getStep()->getText();
		$logMessage = "\t### $step\n";
		HttpLogger::writeLog(HttpLogger::getScenarioLogPath(), $logMessage);
	}

	/**
	 * FIRST AfterScenario HOOK
	 *
	 * NOTE: This method is called after each scenario having the @env-config tag
	 * This ensures that the server is running for clean-up purposes
	 *
	 * @AfterScenario @env-config
	 *
	 * @return void
	 */
	public function startOpencloudServer(): void {
		$response = OcConfigHelper::startOpencloud();
		// 409 is returned if the server is already running
		$this->theHTTPStatusCodeShouldBe([200, 409], 'Starting OpenCloud server', $response);
	}

	/**
	 * Get the externally-defined admin username, if any
	 *
	 * @return string|false
	 */
	private static function getAdminUsernameFromEnvironment() {
		return \getenv('ADMIN_USERNAME');
	}

	/**
	 * Get the externally-defined admin password, if any
	 *
	 * @return string|false
	 */
	private static function getAdminPasswordFromEnvironment() {
		return \getenv('ADMIN_PASSWORD');
	}

	/**
	 * Get the externally-defined regular user password, if any
	 *
	 * @return string|false
	 */
	private static function getRegularUserPasswordFromEnvironment() {
		return \getenv('REGULAR_USER_PASSWORD');
	}

	/**
	 * Get the externally-defined alternate(1) user password, if any
	 *
	 * @return string|false
	 */
	private static function getAlt1UserPasswordFromEnvironment() {
		return \getenv('ALT1_USER_PASSWORD');
	}

	/**
	 * Get the externally-defined alternate(2) user password, if any
	 *
	 * @return string|false
	 */
	private static function getAlt2UserPasswordFromEnvironment() {
		return \getenv('ALT2_USER_PASSWORD');
	}

	/**
	 * Get the externally-defined alternate(3) user password, if any
	 *
	 * @return string|false
	 */
	private static function getAlt3UserPasswordFromEnvironment() {
		return \getenv('ALT3_USER_PASSWORD');
	}

	/**
	 * Get the externally-defined alternate(4) user password, if any
	 *
	 * @return string|false
	 */
	private static function getAlt4UserPasswordFromEnvironment() {
		return \getenv('ALT4_USER_PASSWORD');
	}

	/**
	 * Get the externally-defined sub-admin password, if any
	 *
	 * @return string|false
	 */
	private static function getSubAdminPasswordFromEnvironment() {
		return \getenv('SUB_ADMIN_PASSWORD');
	}

	/**
	 * Get the externally-defined alternate admin password, if any
	 *
	 * @return string|false
	 */
	private static function getAlternateAdminPasswordFromEnvironment() {
		return \getenv('ALTERNATE_ADMIN_PASSWORD');
	}

	/**
	 * Get the externally-defined public link share password, if any
	 *
	 * @return string|false
	 */
	private static function getPublicLinkSharePasswordFromEnvironment() {
		return \getenv('PUBLIC_LINK_SHARE_PASSWORD');
	}

	/**
	 * removes the scheme "http(s)://" (if any) from the front of a URL
	 * note: only needs to handle http or https
	 *
	 * @param string $url
	 *
	 * @return string
	 */
	public function removeSchemeFromUrl(string $url): string {
		$parsedUrl = parse_url($url);
		return $parsedUrl["host"] . ":" . $parsedUrl["port"];
	}

	/**
	 * removes the port from the OpenCloud URL
	 *
	 * @param string $url
	 *
	 * @return string
	 */
	public function removeSchemeAndPortFromUrl(string $url): string {
		$parsedUrl = parse_url($url);
		return $parsedUrl["host"];
	}

	/**
	 * returns the base URL (which is without a slash at the end)
	 *
	 * @return string
	 */
	public function getBaseUrl(): string {
		return $this->baseUrl;
	}

	/**
	 * @return string
	 */
	public function getStorageUsersRoot(): string {
		$ocDataPath = getenv("OC_BASE_DATA_PATH") ? getenv("OC_BASE_DATA_PATH") : getenv("HOME") . '/.opencloud';
		return getenv("STORAGE_USERS_DECOMPOSED_ROOT")
			? getenv("STORAGE_USERS_DECOMPOSED_ROOT")
			: $ocDataPath . "/storage/users";
	}

	/**
	 * returns the path of the base URL
	 * the path is without a slash at the end and without a slash at the beginning
	 *
	 * @return string
	 */
	public function getBasePath(): string {
		$parsedUrl = \parse_url($this->getBaseUrl(), PHP_URL_PATH);
		// If the server-under-test is at the "top" of the domain then parse_url returns null.
		// For example, testing a server at http://localhost:8080 or http://example.com
		if ($parsedUrl === null) {
			$parsedUrl = '';
		}
		return \ltrim($parsedUrl, "/");
	}

	/**
	 * returns the OCS path
	 * the path is without a slash at the end and without a slash at the beginning
	 *
	 * @param string $ocsApiVersion
	 *
	 * @return string
	 */
	public function getOCSPath(string $ocsApiVersion): string {
		return \ltrim($this->getBasePath() . "/ocs/v$ocsApiVersion.php", "/");
	}

	/**
	 * returns the base URL but without "http(s)://" in front of it
	 *
	 * @return string
	 */
	public function getBaseUrlWithoutScheme(): string {
		return $this->removeSchemeFromUrl($this->getBaseUrl());
	}

	/**
	 * returns the base URL but without "http(s)://" and port
	 *
	 * @return string
	 */
	public function getBaseUrlHostName(): string {
		return $this->removeSchemeAndPortFromUrl($this->getBaseUrl());
	}

	/**
	 * returns the base URL but without "http(s)://" and port
	 *
	 * @return string
	 */
	public function getCollaborationHostName(): string {
		return $this->removeSchemeAndPortFromUrl(OcHelper::getCollaborationServiceUrl());
	}

	/**
	 * returns the local base URL (which is without a slash at the end)
	 *
	 * @return string
	 */
	public function getLocalBaseUrl(): string {
		return $this->localBaseUrl;
	}

	/**
	 * returns the local base URL but without "http(s)://" in front of it
	 *
	 * @return string
	 */
	public function getLocalBaseUrlWithoutScheme(): string {
		return $this->removeSchemeFromUrl($this->getLocalBaseUrl());
	}

	/**
	 * returns the remote base URL (which is without a slash at the end)
	 *
	 * @return string
	 */
	public function getRemoteBaseUrl(): string {
		return $this->remoteBaseUrl;
	}

	/**
	 * returns the remote base URL but without "http(s)://" in front of it
	 *
	 * @return string
	 */
	public function getRemoteBaseUrlWithoutScheme(): string {
		return $this->removeSchemeFromUrl($this->getRemoteBaseUrl());
	}

	/**
	 * returns the reference to the current line being executed.
	 *
	 * @return string
	 */
	public function getStepLineRef(): string {
		if (!HttpRequestHelper::sendScenarioLineReferencesInXRequestId()) {
			return '';
		}

		// If we are in BeforeScenario and possibly before any particular step
		// is being executed, then stepLineRef might be empty. In that case
		// return just the string for the scenario.
		if ($this->stepLineRef === '') {
			return $this->scenarioString;
		}
		return $this->stepLineRef;
	}

	/**
	 * returns the base URL without any sub-path e.g. http://localhost:8080
	 *
	 * @return string
	 */
	public function getBaseUrlWithoutPath(): string {
		$parts = \parse_url($this->getBaseUrl());
		$url = $parts ["scheme"] . "://" . $parts["host"];
		if (isset($parts["port"])) {
			$url = "$url:" . $parts["port"];
		}
		return $url;
	}

	/**
	 * @return int
	 */
	public function getOcsApiVersion(): int {
		return $this->ocsApiVersion;
	}

	/**
	 * @return array
	 */
	public function getGuzzleClientHeaders(): array {
		return $this->guzzleClientHeaders;
	}

	/**
	 * @param array $guzzleClientHeaders ['X-Foo' => 'Bar']
	 *
	 * @return void
	 */
	public function setGuzzleClientHeaders(array $guzzleClientHeaders): void {
		$this->guzzleClientHeaders = $guzzleClientHeaders;
	}

	/**
	 * @param array $guzzleClientHeaders ['X-Foo' => 'Bar']
	 *
	 * @return void
	 */
	public function addGuzzleClientHeaders(array $guzzleClientHeaders): void {
		$this->guzzleClientHeaders = \array_merge(
			$this->guzzleClientHeaders,
			$guzzleClientHeaders
		);
	}

	/**
	 * @Given using SharingNG
	 *
	 * @return void
	 */
	public function usingSharingNG(): void {
		$this->useSharingNG = true;
	}

	/**
	 * @Given /^using OCS API version "([^"]*)"$/
	 *
	 * @param string $version
	 *
	 * @return void
	 */
	public function usingOcsApiVersion(string $version): void {
		$this->ocsApiVersion = (int)$version;
	}

	/**
	 * @return string
	 */
	public function getCurrentUser(): string {
		return $this->currentUser;
	}

	/**
	 * @param string $user
	 *
	 * @return void
	 */
	public function setCurrentUser(string $user): void {
		$this->currentUser = $user;
	}

	/**
	 * returns $this->response
	 * some steps use that private var to store the response for other steps
	 *
	 * @return ResponseInterface
	 */
	public function getResponse(): ?ResponseInterface {
		return $this->response;
	}

	/**
	 * let this class remember a response that was received elsewhere
	 * so that steps in this class can be used to examine the response
	 *
	 * @param ResponseInterface|null $response
	 * @param string $username of the user that received the response
	 *
	 * @return void
	 */
	public function setResponse(
		?ResponseInterface $response,
		string             $username = ""
	): void {
		$this->response = $response;
		$this->responseUser = $username;
	}

	/**
	 * @return string
	 */
	public function getCurrentServer(): string {
		return $this->currentServer;
	}

	/**
	 * @Given /^using server "(LOCAL|REMOTE)"$/
	 *
	 * @param string|null $server
	 *
	 * @return string Previous used server
	 */
	public function usingServer(?string $server): string {
		$previousServer = $this->currentServer;
		if ($server === 'LOCAL') {
			$this->baseUrl = $this->localBaseUrl;
			$this->currentServer = 'LOCAL';
		} else {
			$this->baseUrl = $this->remoteBaseUrl;
			$this->currentServer = 'REMOTE';
		}
		return $previousServer;
	}

	/**
	 * @param JsonSchema $schemaObj
	 *
	 * @return void
	 * @throws Exception
	 */
	private function checkInvalidValidator(JsonSchema $schemaObj): void {
		$validators = \array_keys((array)$schemaObj->jsonSerialize());
		foreach ($validators as $validator) {
			Assert::assertContains(
				\ltrim($validator, "$"),
				$this->jsonSchemaValidators,
				"Invalid schema validator: '$validator'"
			);
		}
	}

	/**
	 * Validates against the requirements that object schema should adhere to
	 *
	 * @param JsonSchema $schemaObj
	 *
	 * @return void
	 * @throws Exception
	 */
	public function validateSchemaObject(JsonSchema $schemaObj): void {
		$this->checkInvalidValidator($schemaObj);

		if ($schemaObj->type && $schemaObj->type !== "object") {
			return;
		}

		$notAllowedValidators = ["items", "maxItems", "minItems", "uniqueItems"];

		// check invalid validators
		foreach ($notAllowedValidators as $validator) {
			Assert::assertTrue(null === $schemaObj->$validator, "'$validator' should not be used with object type");
		}

		$propNames = $schemaObj->getPropertyNames();
		$props = $schemaObj->getProperties();
		foreach ($propNames as $propName) {
			$schema = $props->$propName;
			switch ($schema->type) {
				case "array":
					$this->validateSchemaArray($schema);
					break;
				default:
					break;
			}
			// traverse for nested properties
			$this->validateSchemaObject($schema);
		}
	}

	/**
	 * Validates against the requirements that array schema should adhere to
	 *
	 * @param JsonSchema $schemaObj
	 *
	 * @return void
	 * @throws Exception
	 */
	private function validateSchemaArray(JsonSchema $schemaObj): void {
		$this->checkInvalidValidator($schemaObj);

		if ($schemaObj->type && $schemaObj->type !== "array") {
			return;
		}

		$hasTwoElementValidator = ($schemaObj->enum && $schemaObj->const)
		|| ($schemaObj->enum && $schemaObj->items)
		|| ($schemaObj->const && $schemaObj->items);
		Assert::assertFalse($hasTwoElementValidator, "'items', 'enum' and 'const' should not be used together");
		if ($schemaObj->enum || $schemaObj->const) {
			// do not try to validate of enum or const is present
			return;
		}

		$requiredValidators = ["maxItems", "minItems"];
		$optionalValidators = ["items", "uniqueItems"];
		$notAllowedValidators = ["properties", "minProperties", "maxProperties", "required"];
		$errMsg = "'%s' is required for array assertion";

		// check invalid validators
		foreach ($notAllowedValidators as $validator) {
			Assert::assertTrue($schemaObj->$validator === null, "'$validator' should not be used with array type");
		}

		// check required validators
		foreach ($requiredValidators as $validator) {
			Assert::assertNotNull($schemaObj->$validator, \sprintf($errMsg, $validator));
		}

		Assert::assertEquals(
			$schemaObj->minItems,
			$schemaObj->maxItems,
			"'minItems' and 'maxItems' should be equal for strict assertion"
		);

		// check optional validators
		foreach ($optionalValidators as $validator) {
			$value = $schemaObj->$validator;
			switch ($validator) {
				case "items":
					if ($schemaObj->maxItems === 0) {
						break;
					}
					Assert::assertNotNull($schemaObj->$validator, \sprintf($errMsg, $validator));
					if ($schemaObj->maxItems > 1) {
						if (\is_array($value)) {
							foreach ($value as $element) {
								Assert::assertNotNull(
									$element->oneOf,
									"'oneOf' is required to assert more than one elements"
								);
							}
							Assert::fail("'$validator' should be an object not an array");
						}
						Assert::assertFalse(
							$value->allOf || $value->anyOf,
							"'allOf' and 'anyOf' are not allowed in array"
						);
						if ($value->oneOf) {
							Assert::assertNotNull(
								$value->oneOf,
								"'oneOf' is required to assert more than one elements"
							);
							Assert::assertTrue(\is_array($value->oneOf), "'oneOf' should be an array");
							Assert::assertEquals(
								$schemaObj->maxItems,
								\count($value->oneOf),
								"Expected " . $schemaObj->maxItems . " 'oneOf' items but got " . \count($value->oneOf)
							);
						}
					}
					Assert::assertTrue(
						\is_object($value),
						"'$validator' should be an object when expecting 1 element"
					);
					break;
				case "uniqueItems":
					if ($schemaObj->minItems > 1) {
						$errMsg = $value === null ? \sprintf($errMsg, $validator) : "'$validator' should be true";
						Assert::assertTrue($value, $errMsg);
					}
					break;
				default:
					break;
			}
		}

		$items = $schemaObj->items;
		if ($items !== null && $items->oneOf !== null) {
			foreach ($items->oneOf as $oneOfItem) {
				$this->validateSchemaObject($oneOfItem);
			}
		} elseif ($items !== null) {
			$this->validateSchemaObject($items);
		}
	}

	/**
	 * Validates the json schema requirements
	 *
	 * @param JsonSchema $schema
	 *
	 * @return void
	 * @throws Exception
	 */
	public function validateSchemaRequirements(JsonSchema $schema): void {
		Assert::assertNotNull($schema->type, "'type' is required for root level schema");

		switch ($schema->type) {
			case "object":
				$this->validateSchemaObject($schema);
				break;
			case "array":
				$this->validateSchemaArray($schema);
				break;
			default:
				break;
		}
	}

	/**
	 * @param object|array $json
	 * @param object $schema
	 *
	 * @return void
	 * @throws Exception
	 */
	public function assertJsonDocumentMatchesSchema(object|array $json, object $schema): void {
		try {
			$schema = JsonSchema::import($schema);
			$this->validateSchemaRequirements($schema);
			$schema->in($json);
		} catch (JsonSchemaException $e) {
			$this->throwJsonSchemaException($e);
		}
	}

	/**
	 * @param JsonSchemaException $error
	 *
	 * @return array
	 */
	public function getJsonSchemaErrors(JsonSchemaException $error): array {
		$errors = [];
		if (\property_exists($error, "subErrors") && $error->subErrors) {
			foreach ($error->subErrors as $subError) {
				$errors = \array_merge($errors, $this->getJsonSchemaErrors($subError));
			}
		} else {
			$errors[] = $error;
		}
		return $errors;
	}

	/**
	 * @param JsonSchemaException $e
	 *
	 * @return void
	 * @throws Exception
	 */
	public function throwJsonSchemaException(JsonSchemaException $e): void {
		$errors = $this->getJsonSchemaErrors($e);
		$messages = ["JSON Schema validation failed:"];

		$previousPointer = null;
		$errorCount = 0;
		foreach ($errors as $error) {
			$expected = $error->constraint;
			$actual = $error->data;
			$errorMessage = $error->error;
			$schemaPointer = \str_replace("/", ".", \trim($error->getSchemaPointer(), "/"));
			$dataPointer = \str_replace("/", ".", \trim($error->getDataPointer(), "/"));

			$pointer = \str_contains($schemaPointer, "additionalProperties") ? $dataPointer : $schemaPointer;
			if ($pointer === '') {
				$pointer = "{root}";
			}
			if ($pointer === $previousPointer) {
				continue;
			}
			$previousPointer = $pointer;

			$message = ++$errorCount . ". ";
			switch (true) {
				case $error instanceof ArrayException:
				case $error instanceof LogicException:
				case $error instanceof NumericException:
				case $error instanceof StringException:
				case $error instanceof ContentException:
					break;
				case $error instanceof ConstException:
					$errorMessage .= "\n\t   Expected: $expected"
					. "\n\t   Received: $actual";
					break;
				case $error instanceof EnumException:
					$errorMessage .= "\n\t   Expected (One of): " . \join(", ", $expected)
					. "\n\t   Received: $actual";
					break;
				case $error instanceof ObjectException:
					if (\str_starts_with($errorMessage, "Required property missing")) {
						$properties = \join(", ", \array_keys((array)$actual));
						$pointer .= "->required";
						$errorMessage = "Required property missing: id"
						. "\n\t   Received: $properties";
					}
					break;
				case $error instanceof TypeException:
					if (\in_array(\gettype($actual), ['object', 'array'])) {
						$actual = \json_encode($actual, JSON_PRETTY_PRINT);
					}
					break;
				default:
					break;
			}
			$message .= "$pointer:\n\t - $errorMessage\n";
			$messages[] = $message;
		}
		Assert::fail(\join("\n", $messages));
	}

	/**
	 * @When /^user "([^"]*)" sends HTTP method "([^"]*)" to URL "([^"]*)"$/
	 * @When /^user "([^"]*)" tries to send HTTP method "([^"]*)" to URL "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $verb
	 * @param string $url
	 *
	 * @return void
	 */
	public function userSendsHTTPMethodToUrl(string $user, string $verb, string $url): void {
		$user = $this->getActualUsername($user);
		$endpoint = $this->substituteInLineCodes($url, $user);
		$this->setResponse($this->sendingToWithDirectUrl($user, $verb, $endpoint));
	}

	/**
	 * @When the public sends HTTP method :method to URL :url with password :password
	 *
	 * @param string $method
	 * @param string $url
	 * @param string $password
	 *
	 * @return void
	 */
	public function thePublicSendsHttpMethodToUrlWithPassword(string $method, string $url, string $password): void {
		$password = $this->getActualPassword($password);
		$token = $this->shareNgGetLastCreatedLinkShareToken();
		$fullUrl = $this->getBaseUrl() . $url;
		$headers = [
			'Public-Token' => $token
		];
		$this->setResponse(
			HttpRequestHelper::sendRequest(
				$fullUrl,
				$this->getStepLineRef(),
				$method,
				"public",
				$password,
				$headers
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" sends HTTP method "([^"]*)" to URL "([^"]*)" with headers$/
	 *
	 * @param string $user
	 * @param string $verb
	 * @param string $url
	 * @param TableNode $headersTable
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userSendsHTTPMethodToUrlWithHeaders(
		string $user,
		string $verb,
		string $url,
		TableNode $headersTable
	): void {
		$this->verifyTableNodeColumns(
			$headersTable,
			['header', 'value']
		);

		$user = $this->getActualUsername($user);
		$url = $this->substituteInLineCodes($url, $user);
		$url = "/" . \ltrim(\str_replace($this->getBaseUrl(), "", $url), "/");

		$headers = [];
		foreach ($headersTable as $row) {
			$headers[$row['header']] = $row['value'];
		}
		$response = $this->sendingToWithDirectUrl($user, $verb, $url, null, null, $headers);
		$this->setResponse($response);
	}

	/**
	 * This function is needed to use a vertical fashion in the gherkin tables.
	 *
	 * @param array $arrayOfArrays
	 *
	 * @return array
	 */
	public function simplifyArray(array $arrayOfArrays): array {
		$a = \array_map(
			function ($subArray) {
				return $subArray[0];
			},
			$arrayOfArrays
		);
		return $a;
	}

	/**
	 * @When user :user sends HTTP method :method to URL :davPath with content :content
	 *
	 * @param string $user
	 * @param string $method
	 * @param string $davPath
	 * @param string $content
	 *
	 * @return void
	 */
	public function userSendsHttpMethodToUrlWithContent(
		string $user,
		string $method,
		string $davPath,
		string $content
	): void {
		$this->setResponse($this->sendingToWithDirectUrl($user, $method, $davPath, $content));
	}

	/**
	 * @When /^user "([^"]*)" sends HTTP method "([^"]*)" to URL "([^"]*)" with password "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $verb
	 * @param string $url
	 * @param string $password
	 *
	 * @return void
	 */
	public function userSendsHTTPMethodToUrlWithPassword(
		string $user,
		string $verb,
		string $url,
		string $password
	): void {
		$this->setResponse($this->sendingToWithDirectUrl($user, $verb, $url, null, $password));
	}

	/**
	 * @param string $user
	 * @param string $verb
	 * @param string $url
	 * @param string|null $body
	 * @param string|null $password
	 * @param array|null $headers
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function sendingToWithDirectUrl(
		string $user,
		string $verb,
		string $url,
		?string $body = null,
		?string $password = null,
		?array $headers = null
	): ResponseInterface {
		$url = \ltrim($url, '/');
		if (WebdavHelper::isDAVRequest($url)) {
			$url = WebdavHelper::prefixRemotePhp($url);
		}
		$fullUrl = $this->getBaseUrl() . "/$url";

		if ($password === null) {
			$password = $this->getPasswordForUser($user);
		}

		$reqHeaders = $this->guzzleClientHeaders;

		$config = null;
		if ($this->sourceIpAddress !== null) {
			$config = [
				'curl' => [
					CURLOPT_INTERFACE => $this->sourceIpAddress
				]
			];
		}

		$cookies = null;
		if (!empty($this->cookieJar->toArray())) {
			$cookies = $this->cookieJar;
		}

		if (isset($this->requestToken)) {
			$reqHeaders['requesttoken'] = $this->requestToken;
		}

		if ($headers) {
			$reqHeaders = \array_merge($headers, $reqHeaders);
		}

		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->getStepLineRef(),
			$verb,
			$user,
			$password,
			$reqHeaders,
			$body,
			$config,
			$cookies
		);
	}

	/**
	 * @param string $url
	 *
	 * @return bool
	 */
	public function isAPublicLinkUrl(string $url): bool {
		if (OcHelper::isTestingOnReva()) {
			$urlEnding = \ltrim($url, '/');
		} else {
			if (\substr($url, 0, 4) !== "http") {
				return false;
			}
			$urlEnding = \substr($url, \strlen($this->getBaseUrl() . '/'));
		}

		$matchResult = \preg_match("%^(#/)?s/([a-zA-Z0-9]{15})$%", $urlEnding);

		// preg_match returns (int) 1 for a match, we want to return a boolean.
		if ($matchResult === 1) {
			$isPublicLinkUrl = true;
		} else {
			$isPublicLinkUrl = false;
		}
		return $isPublicLinkUrl;
	}

	/**
	 * Check that the status code in the saved response is the expected status
	 * code, or one of the expected status codes.
	 *
	 * @param int|int[]|string|string[] $expectedStatusCode
	 * @param string|null $message
	 * @param ResponseInterface|null $response
	 *
	 * @return void
	 */
	public function theHTTPStatusCodeShouldBe(
		$expectedStatusCode,
		?string $message = "",
		?ResponseInterface $response = null
	): void {
		$response = $response ?? $this->response;
		$actualStatusCode = $response->getStatusCode();
		if (\is_array($expectedStatusCode)) {
			if ($message === "") {
				$message = "HTTP status code $actualStatusCode is not one of the expected values "
				. \implode(" or ", $expectedStatusCode);
			}

			Assert::assertContainsEquals(
				$actualStatusCode,
				$expectedStatusCode,
				$message
			);
		} else {
			if ($message === "") {
				$message = "HTTP status code $actualStatusCode is not the expected value $expectedStatusCode";
			}

			Assert::assertEquals(
				$expectedStatusCode,
				$actualStatusCode,
				$message
			);
		}
	}

	/**
	 * @param PyStringNode|string $schemaString
	 *
	 * @return mixed
	 */
	public function getJSONSchema($schemaString) {
		if (\gettype($schemaString) !== 'string') {
			$schemaString = $schemaString->getRaw();
		}
		$schemaString = $this->substituteInLineCodes($schemaString);
		$schema = \json_decode($schemaString);
		Assert::assertNotNull($schema, 'schema is not valid JSON');
		return $schema;
	}

	/**
	 * returns json decoded body content of a json response as an object
	 *
	 * @param ResponseInterface|null $response
	 *
	 * @return mixed
	 */
	public function getJsonDecodedResponseBodyContent(ResponseInterface $response = null): mixed {
		$response = $response ?? $this->response;
		$response->getBody()->rewind();
		return HttpRequestHelper::getJsonDecodedResponseBodyContent($response);
	}

	/**
	 * @Then the ocs JSON data of the response should match
	 *
	 * @param PyStringNode $schemaString
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function theOcsDataOfTheResponseShouldMatch(
		PyStringNode $schemaString
	): void {
		$jsonResponse = $this->getJsonDecodedResponseBodyContent();
		$this->assertJsonDocumentMatchesSchema(
			$jsonResponse->ocs->data,
			$this->getJSONSchema($schemaString)
		);
	}

	/**
	 * @Then the JSON data of the response should match
	 *
	 * @param PyStringNode $schemaString
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theJsonDataOfTheResponseShouldMatch(PyStringNode $schemaString): void {
		$responseBody = $this->getJsonDecodedResponseBodyContent();
		$this->assertJsonDocumentMatchesSchema(
			$responseBody,
			$this->getJSONSchema($schemaString)
		);
	}

	/**
	 * @Then /^the HTTP status code should be "([^"]*)"$/
	 *
	 * @param int|string $statusCode
	 *
	 * @return void
	 */
	public function thenTheHTTPStatusCodeShouldBe($statusCode): void {
		$this->theHTTPStatusCodeShouldBe($statusCode);
	}

	/**
	 * @Then /^the HTTP status code should be "([^"]*)" or "([^"]*)"$/
	 *
	 * @param int|string $statusCode1
	 * @param int|string $statusCode2
	 *
	 * @return void
	 */
	public function theHTTPStatusCodeShouldBeOr($statusCode1, $statusCode2): void {
		$this->theHTTPStatusCodeShouldBe(
			[$statusCode1, $statusCode2]
		);
	}

	/**
	 * @Then /^the HTTP status code should be between "(\d+)" and "(\d+)"$/
	 *
	 * @param int|string $minStatusCode
	 * @param int|string $maxStatusCode
	 * @param ResponseInterface|null $response
	 *
	 * @return void
	 */
	public function theHTTPStatusCodeShouldBeBetween(
		$minStatusCode,
		$maxStatusCode,
		?ResponseInterface $response= null
	): void {
		$response = $response ?? $this->response;
		$statusCode = $response->getStatusCode();
		$message = "The HTTP status code $statusCode is not between $minStatusCode and $maxStatusCode";
		Assert::assertGreaterThanOrEqual(
			$minStatusCode,
			$statusCode,
			$message
		);
		Assert::assertLessThanOrEqual(
			$maxStatusCode,
			$statusCode,
			$message
		);
	}

	/**
	 * @param string $path
	 * @param string $filename
	 *
	 * @return void
	 */
	public static function removeFile(string $path, string $filename): void {
		if (\file_exists("$path$filename")) {
			\unlink("$path$filename");
		}
	}

	/**
	 * Creates a file locally in the file system of the test runner
	 * The file will be available to upload to the server
	 *
	 * @param string $name
	 * @param string $size
	 * @param string $endData
	 *
	 * @return void
	 */
	public function createLocalFileOfSpecificSize(string $name, string $size, string $endData = 'a'): void {
		$folder = $this->workStorageDirLocation();
		if (!\is_dir($folder)) {
			\mkDir($folder);
		}
		$file = \fopen($folder . $name, 'w');
		\fseek($file, $size - \strlen($endData), SEEK_CUR);
		\fwrite($file, $endData); // write the end data to force the file size
		\fclose($file);
	}

	/**
	 * Make a directory under the server root on the OpenCloud server
	 *
	 * @param string $dirPathFromServerRoot e.g. 'apps2/myapp/appinfo'
	 *
	 * @return void
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public function mkDirOnServer(string $dirPathFromServerRoot): void {
		SetupHelper::mkDirOnServer(
			$dirPathFromServerRoot,
			$this->getStepLineRef(),
			$this->getBaseUrl(),
			$this->getAdminUsername(),
			$this->getAdminPassword()
		);
	}

	/**
	 * @return string
	 */
	public function getAdminUsername(): string {
		return $this->adminUsername;
	}

	/**
	 * @return string
	 */
	public function getAdminPassword(): string {
		return $this->adminPassword;
	}

	/**
	 * @param string|null $userName
	 *
	 * @return string
	 */
	public function getPasswordForUser(?string $userName): string {
		$userNameNormalized = $this->normalizeUsername($userName);
		$username = $this->getActualUsername($userNameNormalized);
		if ($username === $this->getAdminUsername()) {
			return $this->getAdminPassword();
		} elseif (\array_key_exists($username, $this->createdUsers)) {
			return (string)$this->createdUsers[$username]['password'];
		} elseif (\array_key_exists($username, $this->createdRemoteUsers)) {
			return (string)$this->createdRemoteUsers[$username]['password'];
		}

		// The user has not been created yet, see if there is a replacement
		// defined for the user.
		$usernameReplacements = $this->usersToBeReplaced();
		if (isset($usernameReplacements)) {
			if (isset($usernameReplacements[$userNameNormalized])) {
				return $usernameReplacements[$userNameNormalized]['password'];
			}
		}

		// Fall back to the default password used for the well-known users.
		if ($username === 'regularuser') {
			return $this->regularUserPassword;
		} elseif ($username === 'alice') {
			return $this->regularUserPassword;
		} elseif ($username === 'brian') {
			return $this->alt1UserPassword;
		} elseif ($username === 'carol') {
			return $this->alt2UserPassword;
		} elseif ($username === 'david') {
			return $this->alt3UserPassword;
		} elseif ($username === 'emily') {
			return $this->alt4UserPassword;
		} elseif ($username === 'usergrp') {
			return $this->regularUserPassword;
		} elseif ($username === 'sharee1') {
			return $this->regularUserPassword;
		}

		// The user has not been created yet and is not one of the pre-known
		// users. So let the caller have the default password.
		return (string)$this->getActualPassword($this->regularUserPassword);
	}

	/**
	 * @param string $username
	 * @param string $password
	 *
	 * @return void
	 * @throws Exception
	 */
	public function updateUserPassword(string $username, string $password): void {
		$username = $this->normalizeUsername($username);
		if ($username === $this->getAdminUsername()) {
			$this->adminPassword = $password;
		} elseif (\array_key_exists($username, $this->createdUsers)) {
			$this->createdUsers[$username]['password'] = $password;
		} else {
			throw new Exception("User '$username' not found");
		}
	}

	/**
	 * Get the display name of the user.
	 *
	 * For users that have already been created, return their display name.
	 * For special known usernames, return the display name that is also used by LDAP tests.
	 * For other users, return null. They will not be assigned any particular
	 * display name by this function.
	 *
	 * @param string $userName
	 *
	 * @return string|null
	 */
	public function getDisplayNameForUser(string $userName): ?string {
		$userNameNormalized = $this->normalizeUsername($userName);
		$username = $this->getActualUsername($userNameNormalized);
		if (\array_key_exists($username, $this->createdUsers)) {
			if (isset($this->createdUsers[$username]['displayname'])) {
				return (string)$this->createdUsers[$username]['displayname'];
			}
			return $userName;
		}
		if (\array_key_exists($username, $this->createdRemoteUsers)) {
			if (isset($this->createdRemoteUsers[$username]['displayname'])) {
				return (string)$this->createdRemoteUsers[$username]['displayname'];
			}
			return $userName;
		}

		// The user has not been created yet, see if there is a replacement
		// defined for the user.
		$usernameReplacements = $this->usersToBeReplaced();
		if (isset($usernameReplacements)) {
			if (isset($usernameReplacements[$userNameNormalized])) {
				return $usernameReplacements[$userNameNormalized]['displayname'];
			} elseif (isset($usernameReplacements[$userName])) {
				return $usernameReplacements[$userName]['displayname'];
			}
		}

		// Fall back to the default display name used for the well-known users.
		if ($username === 'regularuser') {
			return 'Regular User';
		} elseif ($username === 'alice') {
			return 'Alice Hansen';
		} elseif ($username === 'brian') {
			return 'Brian Murphy';
		} elseif ($username === 'carol') {
			return 'Carol King';
		} elseif ($username === 'david') {
			return 'David Lopez';
		} elseif ($username === 'emily') {
			return 'Emily Wagner';
		} elseif ($username === 'usergrp') {
			return 'User Grp';
		} elseif ($username === 'sharee1') {
			return 'Sharee One';
		} elseif ($username === 'sharee2') {
			return 'Sharee Two';
		} elseif (\in_array($username, ["grp1", "***redacted***"])) {
			return $username;
		}
		return null;
	}

	/**
	 * Get the email address of the user.
	 *
	 * For users that have already been created, return their email address.
	 * For special known usernames, return the email address that is also used by LDAP tests.
	 * For other users, return null. They will not be assigned any particular
	 * email address by this function.
	 *
	 * @param string $userName
	 *
	 * @return string|null
	 */
	public function getEmailAddressForUser(string $userName): ?string {
		$userNameNormalized = $this->normalizeUsername($userName);
		$username = $this->getActualUsername($userNameNormalized);
		if (\array_key_exists($username, $this->createdUsers)) {
			return (string)$this->createdUsers[$username]['email'];
		}
		if (\array_key_exists($username, $this->createdRemoteUsers)) {
			return (string)$this->createdRemoteUsers[$username]['email'];
		}

		// The user has not been created yet, see if there is a replacement
		// defined for the user.
		$usernameReplacements = $this->usersToBeReplaced();
		if (isset($usernameReplacements)) {
			if (isset($usernameReplacements[$userNameNormalized])) {
				return $usernameReplacements[$userNameNormalized]['email'];
			} elseif (isset($usernameReplacements[$userName])) {
				return $usernameReplacements[$userName]['email'];
			}
		}

		// Fall back to the default display name used for the well-known users.
		if ($username === 'regularuser') {
			return 'regularuser@example.org';
		} elseif ($username === 'alice') {
			return 'alice@example.org';
		} elseif ($username === 'brian') {
			return 'brian@example.org';
		} elseif ($username === 'carol') {
			return 'carol@example.org';
		} elseif ($username === 'david') {
			return 'david@example.org';
		} elseif ($username === 'emily') {
			return 'emily@example.org';
		} elseif ($username === 'usergrp') {
			return 'usergrp@example.org';
		} elseif ($username === 'sharee1') {
			return 'sharee1@example.org';
		} else {
			return null;
		}
	}

	/**
	 * @param string|null $functionalUsername
	 *
	 * @return string|null
	 * @throws JsonException
	 */
	public function getActualUsername(?string $functionalUsername): ?string {
		if ($functionalUsername === null) {
			return null;
		}
		$usernames = $this->usersToBeReplaced();
		if (isset($usernames)) {
			if (isset($usernames[$functionalUsername])) {
				return $usernames[$functionalUsername]['username'];
			}
			$normalizedUsername = $this->normalizeUsername($functionalUsername);
			if (isset($usernames[$normalizedUsername])) {
				return $usernames[$normalizedUsername]['username'];
			}
		}
		if ($functionalUsername === "%admin%") {
			return $this->getAdminUsername();
		}
		return $functionalUsername;
	}

	/**
	 * @param string|null $functionalPassword
	 *
	 * @return string|null
	 */
	public function getActualPassword(?string $functionalPassword): ?string {
		if ($functionalPassword === "%regular%") {
			return $this->regularUserPassword;
		} elseif ($functionalPassword === "%alt1%") {
			return $this->alt1UserPassword;
		} elseif ($functionalPassword === "%alt2%") {
			return $this->alt2UserPassword;
		} elseif ($functionalPassword === "%alt3%") {
			return $this->alt3UserPassword;
		} elseif ($functionalPassword === "%alt4%") {
			return $this->alt4UserPassword;
		} elseif ($functionalPassword === "%subadmin%") {
			return $this->subAdminPassword;
		} elseif ($functionalPassword === "%admin%") {
			return $this->getAdminPassword();
		} elseif ($functionalPassword === "%altadmin%") {
			return $this->alternateAdminPassword;
		} elseif ($functionalPassword === "%public%") {
			return $this->publicLinkSharePassword;
		} elseif ($functionalPassword === "%remove%") {
			return "";
		} else {
			return $functionalPassword;
		}
	}

	/**
	 * @When the administrator requests status.php
	 *
	 * @return void
	 */
	public function theAdministratorRequestsStatusPhp(): void {
		$this->response = $this->getStatusPhp();
	}

	/**
	 *
	 * @return ResponseInterface
	 */
	public function getStatusPhp(): ResponseInterface {
		$fullUrl = $this->getBaseUrl() . "/status.php";

		$config = null;
		if ($this->sourceIpAddress !== null) {
			$config = [
				'curl' => [
					CURLOPT_INTERFACE => $this->sourceIpAddress
				]
			];
		}

		return HttpRequestHelper::get(
			$fullUrl,
			$this->getStepLineRef(),
			$this->getAdminUsername(),
			$this->getAdminPassword(),
			$this->guzzleClientHeaders,
			null,
			$config
		);
	}

	/**
	 * @param ResponseInterface|null $response
	 *
	 * @return array
	 */
	public function getJsonDecodedResponse(?ResponseInterface $response = null): array {
		if ($response === null) {
			$response = $this->getResponse();
		}
		return \json_decode(
			(string)$response->getBody(),
			true
		);
	}

	/**
	 *
	 * @return array
	 */
	public function getJsonDecodedStatusPhp(): array {
		return $this->getJsonDecodedResponse(
			$this->getStatusPhp()
		);
	}

	/**
	 * @return string
	 */
	public function getEditionFromStatus(): string {
		$decodedResponse = $this->getJsonDecodedStatusPhp();
		if (isset($decodedResponse['edition'])) {
			return $decodedResponse['edition'];
		}
		return '';
	}

	/**
	 * @return string|null
	 */
	public function getProductNameFromStatus(): ?string {
		$decodedResponse = $this->getJsonDecodedStatusPhp();
		if (isset($decodedResponse['productname'])) {
			return $decodedResponse['productname'];
		}
		return '';
	}

	/**
	 * @return string|null
	 */
	public function getVersionFromStatus(): ?string {
		$decodedResponse = $this->getJsonDecodedStatusPhp();
		if (isset($decodedResponse['version'])) {
			return $decodedResponse['version'];
		}
		return '';
	}

	/**
	 * @return string|null
	 */
	public function getVersionStringFromStatus(): ?string {
		$decodedResponse = $this->getJsonDecodedStatusPhp();
		if (isset($decodedResponse['versionstring'])) {
			return $decodedResponse['versionstring'];
		}
		return '';
	}

	/**
	 * returns a string that can be used to check a URL of comments with
	 * regular expression (without delimiter)
	 *
	 * @return string
	 */
	public function getCommentUrlRegExp(): string {
		$basePath = \ltrim($this->getBasePath() . "/", "/");
		$commentsPath = WebDAVHelper::getDavPath(WebDavHelper::DAV_VERSION_NEW, null, "comments");
		return "/$basePath/$commentsPath/([0-9]+)";
	}

	/**
	 * substitutes codes like %base_url% with the value
	 * if the given value does not have anything to be substituted
	 * then it is returned unmodified
	 *
	 * @param string|null $value
	 * @param string|null $user
	 * @param array|null $functions associative array of functions and parameters to be
	 *                              called on every replacement string before the
	 *                              replacement
	 *                              function name has to be the key and the parameters an
	 *                              own array
	 *                              the replacement itself will be used as first parameter
	 *                              e.g. substituteInLineCodes($value, ['preg_quote' => ['/']])
	 * @param array|null $additionalSubstitutions
	 *                         array of additional substitution configurations
	 *                           [
	 *                             [
	 *                               "code" => "%my_code%",
	 *                               "function" => [
	 *                                                $myClass,
	 *                                                "myFunction"
	 *                               ],
	 *                               "parameter" => []
	 *                             ],
	 *                           ]
	 * @param string|null $group
	 * @param string|null $userName
	 *
	 * @return string
	 */
	public function substituteInLineCodes(
		?string $value,
		?string $user = null,
		?array  $functions = [],
		?array  $additionalSubstitutions = [],
		?string $group = null,
		?string $userName = null
	): ?string {
		$substitutions = [
			[
				"code" => "%base_url%",
				"function" => [
					$this,
					"getBaseUrl"
				],
				"parameter" => []
			],
			[
				"code" => "%local_base_url%",
				"function" => [
					$this,
					"getLocalBaseUrl"
				],
				"parameter" => []
			],
			[
				"code" => "%remote_base_url%",
				"function" => [
					$this,
					"getRemoteBaseUrl"
				],
				"parameter" => []
			],
			[
				"code" => "%base_host_port%",
				"function" => [
					$this,
					"getBaseUrlWithoutScheme"
				],
				"parameter" => []
			],
			[
				"code" => "%local_host_port%",
				"function" => [
					$this,
					"getLocalBaseUrlWithoutScheme"
				],
				"parameter" => []
			],
			[
				"code" => "%remote_host_port%",
				"function" => [
					$this,
					"getRemoteBaseUrlWithoutScheme"
				],
				"parameter" => []
			],
			[
				"code" => "%storage_path%",
				"function" => [
					$this,
					"getStorageUsersRoot"
				],
				"parameter" => []
			],
			[
				"code" => "%base_url_without_scheme%",
				"function" => [
					$this,
					"getBaseUrlWithoutScheme"
				],
				"parameter" => []
			],
			[
				"code" => "%base_url_hostname%",
				"function" => [
					$this,
					"getBaseUrlHostName"
				],
				"parameter" => []
			],
			[
				"code" => "%collaboration_hostname%",
				"function" => [
					$this,
					"getCollaborationHostName"
				],
				"parameter" => []
			],
			[
				"code" => "%remote_server%",
				"function" => [
					$this,
					"getRemoteBaseUrl"
				],
				"parameter" => []
			],
			[
				"code" => "%remote_server_without_scheme%",
				"function" => [
					$this,
					"getRemoteBaseUrlWithoutScheme"
				],
				"parameter" => []
			],
			[
				"code" => "%local_server%",
				"function" => [
					$this,
					"getLocalBaseUrl"
				],
				"parameter" => []
			],
			[
				"code" => "%local_server_without_scheme%",
				"function" => [
					$this,
					"getLocalBaseUrlWithoutScheme"
				],
				"parameter" => []
			],
			[
				"code" => "%base_path%",
				"function" => [
					$this,
					"getBasePath"
				],
				"parameter" => []
			],
			[
				"code" => "%ocs_path_v1%",
				"function" => [
					$this,
					"getOCSPath"
				],
				"parameter" => ["1"]
			],
			[
				"code" => "%ocs_path_v2%",
				"function" => [
					$this,
					"getOCSPath"
				],
				"parameter" => ["2"]
			],
			[
				"code" => "%productname%",
				"function" => [
					$this,
					"getProductNameFromStatus"
				],
				"parameter" => []
			],
			[
				"code" => "%edition%",
				"function" => [
					$this,
					"getEditionFromStatus"
				],
				"parameter" => []
			],
			[
				"code" => "%version%",
				"function" => [
					$this,
					"getVersionFromStatus"
				],
				"parameter" => []
			],
			[
				"code" => "%versionstring%",
				"function" => [
					$this,
					"getVersionStringFromStatus"
				],
				"parameter" => []
			],
			[
				"code" => "%a_comment_url%",
				"function" => [
					$this,
					"getCommentUrlRegExp"
				],
				"parameter" => []
			],
			[
				"code" => "%last_share_id%",
				"function" => [
					$this,
					"getLastCreatedUserGroupShareId"
				],
				"parameter" => []
			],
			[
				"code" => "%last_public_share_token%",
				"function" => [
					$this,
					"getLastCreatedPublicShareToken"
				],
				"parameter" => []
			],
			[
				"code" => "%user_id%",
				"function" => [
					$this, "getAttributeOfCreatedUser"
				],
					"parameter" => [$userName, 'id']
				],
			[
				"code" => "%group_id%",
				"function" => [
					$this, "getGroupIdByGroupName"
				],
				"parameter" => [$group]
			],
			[
				"code" => "%user_id_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\GraphHelper',
					"getUUIDv4Regex"
				],
				"parameter" => []
			],
			[
				"code" => "%federated_user_id_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\GraphHelper',
					"getFederatedUserRegex"
				],
				"parameter" => []
			],
			[
				"code" => "%federated_file_id_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\GraphHelper',
					"getFederatedFileIdRegex"
				],
				"parameter" => []
			],
			[
				"code" => "%group_id_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\GraphHelper',
					"getUUIDv4Regex"
				],
				"parameter" => []
			],
			[
				"code" => "%role_id_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\GraphHelper',
					"getUUIDv4Regex"
				],
				"parameter" => []
			],
			[
				"code" => "%permissions_id_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\GraphHelper',
					"getPermissionsIdRegex"
				],
				"parameter" => []
			],
			[
				"code" => "%file_id_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\GraphHelper',
					"getFileIdRegex"
				],
				"parameter" => []
			],
			[
				"code" => "%space_id_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\GraphHelper',
					"getSpaceIdRegex"
				],
				"parameter" => []
			],
			[
				"code" => "%share_id_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\GraphHelper',
					"getShareIdRegex"
				],
				"parameter" => []
			],
			[
				"code" => "%etag_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\GraphHelper',
					"getEtagRegex"
				],
				"parameter" => []
			],
			[
				"code" => "%tus_upload_location%",
				"function" => [
					$this->tusContext,
					"getLastTusResourceLocation"
				],
				"parameter" => []
			],
			[
				"code" => "%fed_invitation_token%",
				"function" => [
					$this->ocmContext,
					"getLastFederatedInvitationToken"
				],
				"parameter" => []
			],
			[
				"code" => "%uuidv4_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\GraphHelper',
					"getUUIDv4Regex"
				],
				"parameter" => []
			],
			[
				"code" => "%request_id_pattern%",
				"function" => [
					__NAMESPACE__ . '\TestHelpers\HttpRequestHelper',
					"getXRequestIdRegex"
				],
				"parameter" => []
			]
		];
		if ($user !== null) {
			array_push(
				$substitutions,
				[
					"code" => "%username%",
					"function" => [
						$this,
						"getActualUsername"
					],
					"parameter" => [$user]
				],
				[
					"code" => "%displayname%",
					"function" => [
						$this,
						"getDisplayNameForUser"
					],
					"parameter" => [$user]
				],
				[
					"code" => "%password%",
					"function" => [
						$this,
						"getPasswordForUser"
					],
					"parameter" => [$user]
				],
				[
					"code" => "%emailaddress%",
					"function" => [
						$this,
						"getEmailAddressForUser"
					],
					"parameter" => [$user]
				],
				[
					"code" => "%spaceid%",
					"function" => [
						$this,
						"getPersonalSpaceIdForUser",
					],
					"parameter" => [$user, true]
				],
				[
				"code" => "%user_id%",
				"function" =>
				[$this, "getAttributeOfCreatedUser"],
				"parameter" => [$userName, 'id']
				],
				[
				"code" => "%group_id%",
				"function" =>
				[$this, "getGroupIdByGroupName"],
				"parameter" => [$group]
				]
			);

			if (!OcHelper::isTestingOnReva()) {
				array_push(
					$substitutions,
					[
						"code" => "%shares_drive_id%",
						"function" => [
							$this->spacesContext,
							"getSpaceIdByName"
						],
						"parameter" => [$user, "Shares"]
					]
				);
			}
		}

		if (!empty($additionalSubstitutions)) {
			$substitutions = \array_merge($substitutions, $additionalSubstitutions);
		}

		foreach ($substitutions as $substitution) {
			if (strpos($value, $substitution['code']) === false) {
				continue;
			}

			$replacement = \call_user_func_array(
				$substitution["function"],
				$substitution["parameter"]
			);

			// do not run functions on regex patterns
			if (!\str_ends_with($value, "_pattern%")) {
				foreach ($functions as $function => $parameters) {
					$replacement = \call_user_func_array(
						$function,
						\array_merge([$replacement], $parameters)
					);
				}
			}
			$value = \str_replace(
				$substitution["code"],
				$replacement,
				$value
			);
		}
		return $value;
	}

	/**
	 * returns personal space id for user if the test is using the spaces dav path
	 * or if alwaysDoIt is set to true,
	 * otherwise it returns null.
	 *
	 * @param string $user
	 * @param bool $alwaysDoIt default false. Set to true
	 *
	 * @return string|null
	 * @throws GuzzleException
	 */
	public function getPersonalSpaceIdForUser(string $user, bool $alwaysDoIt = false): ?string {
		if ($alwaysDoIt || ($this->getDavPathVersion() === WebDavHelper::DAV_VERSION_SPACES)) {
			return WebDavHelper::getPersonalSpaceIdForUserOrFakeIfNotFound(
				$this->getBaseUrl(),
				$user,
				$this->getPasswordForUser($user),
				$this->getStepLineRef()
			);
		}
		return null;
	}

	/**
	 * @return string
	 */
	public function temporaryStorageSubfolderName(): string {
		return "work_tmp";
	}

	/**
	 * @return string
	 */
	public function acceptanceTestsDirLocation(): string {
		return \dirname(__FILE__) . "/../";
	}

	/**
	 * @return string
	 */
	public function workStorageDirLocation(): string {
		return $this->acceptanceTestsDirLocation() . $this->temporaryStorageSubfolderName() . "/";
	}

	/**
	 * This will run before EVERY scenario.
	 * It will set the properties for this object.
	 *
	 * @BeforeScenario
	 *
	 * @param BeforeScenarioScope $scope
	 *
	 * @return void
	 * @throws Exception
	 */
	public function before(BeforeScenarioScope $scope): void {
		$this->scenarioStartTime = \time();

		// Get the environment
		$environment = $scope->getEnvironment();
		// Get all the contexts you need in this context
		$this->ocsContext = BehatHelper::getContext($scope, $environment, 'OCSContext');
		$this->authContext = BehatHelper::getContext($scope, $environment, 'AuthContext');
		$this->tusContext = BehatHelper::getContext($scope, $environment, 'TUSContext');
		$this->ocmContext = BehatHelper::getContext($scope, $environment, 'OcmContext');
		$this->graphContext = BehatHelper::getContext($scope, $environment, 'GraphContext');
		$this->spacesContext = BehatHelper::getContext($scope, $environment, 'SpacesContext');

		$scenarioLine = $scope->getScenario()->getLine();
		$featureFile = $scope->getFeature()->getFile();
		$suiteName = $scope->getSuite()->getName();
		$featureFileName = \basename($featureFile);
		if (HttpRequestHelper::sendScenarioLineReferencesInXRequestId()) {
			$this->scenarioString = $suiteName . '/' . $featureFileName . ':' . $scenarioLine;
		}

		// Initialize SetupHelper
		SetupHelper::init(
			$this->getAdminUsername(),
			$this->getAdminPassword(),
			$this->getBaseUrl(),
		);

		if ($this->isTestingWithLdap()) {
			$suiteParameters = SetupHelper::getSuiteParameters($scope);
			$this->connectToLdap($suiteParameters);
		}
	}

	/**
	 * This will run before EVERY step.
	 *
	 * @BeforeStep
	 *
	 * @param BeforeStepScope $scope
	 *
	 * @return void
	 */
	public function beforeEachStep(BeforeStepScope $scope): void {
		if (HttpRequestHelper::sendScenarioLineReferencesInXRequestId()) {
			$this->stepLineRef = $this->scenarioString . '-' . $scope->getStep()->getLine();
		} else {
			$this->stepLineRef = '';
		}
	}

	/**
	 * @AfterScenario
	 *
	 * @return void
	 */
	public function restoreAdminPassword(): void {
		if ($this->adminPassword !== $this->originalAdminPassword) {
			$this->resetUserPasswordAsAdminUsingTheProvisioningApi(
				$this->getAdminUsername(),
				$this->originalAdminPassword
			);
			$this->adminPassword = $this->originalAdminPassword;
		}
	}

	/**
	 * @AfterScenario
	 *
	 * @return void
	 */
	public function deleteAllResourceCreatedByAdmin(): void {
		foreach ($this->adminResources as $resource) {
			$this->deleteFile("admin", $resource);
		}
	}

	/**
	 * @AfterScenario
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function cleanDataAfterTests(): void {
		if (!OcHelper::isTestingOnReva() && !OcHelper::isUsingPreparedLdapUsers()) {
			$this->spacesContext->deleteAllProjectSpaces();
		}

		if (OcHelper::isTestingOnReva()) {
			OcHelper::deleteRevaUserData($this->getCreatedUsers());
		}
		if ($this->isTestingWithLdap()) {
			$this->deleteLdapUsersAndGroups();
		}
		$this->cleanupDatabaseUsers();
		$this->cleanupDatabaseGroups();
	}

	/**
	 * @BeforeScenario @temporary_storage_on_server
	 *
	 * @return void
	 * @throws Exception
	 */
	public function makeTemporaryStorageOnServerBefore(): void {
		$this->mkDirOnServer(
			TEMPORARY_STORAGE_DIR_ON_REMOTE_SERVER
		);
	}

	/**
	 * @AfterScenario @temporary_storage_on_server
	 *
	 * @return void
	 * @throws Exception
	 */
	public function removeTemporaryStorageOnServerAfter(): void {
		SetupHelper::rmDirOnServer(
			TEMPORARY_STORAGE_DIR_ON_REMOTE_SERVER,
			$this->getStepLineRef()
		);
	}

	/**
	 * @AfterScenario
	 *
	 * @return void
	 */
	public function removeCreatedFilesAfter(): void {
		foreach ($this->createdFiles as $file) {
			\unlink($file);
		}
	}

	/**
	 * @AfterScenario
	 *
	 * clear space id reference
	 *
	 * @return void
	 * @throws Exception
	 */
	public function clearSpaceId(): void {
		if (\count(WebDavHelper::$spacesIdRef) > 0) {
			WebDavHelper::$spacesIdRef = [];
		}
	}

	/**
	 * Verify that the tableNode contains expected headers
	 *
	 * @param TableNode|null $table
	 * @param array|null $requiredHeader
	 * @param array|null $allowedHeader
	 *
	 * @return void
	 * @throws Exception
	 */
	public function verifyTableNodeColumns(
		?TableNode $table,
		?array $requiredHeader = [],
		?array $allowedHeader = []
	): void {
		if ($table === null || \count($table->getHash()) < 1) {
			throw new Exception("Table should have at least one row.");
		}
		$tableHeaders = $table->getRows()[0];
		$allowedHeader = \array_unique(\array_merge($requiredHeader, $allowedHeader));
		if ($requiredHeader != []) {
			foreach ($requiredHeader as $element) {
				if (!\in_array($element, $tableHeaders)) {
					throw new Exception("Row with header '$element' expected to be in table but not found");
				}
			}
		}

		if ($allowedHeader != []) {
			foreach ($tableHeaders as $element) {
				if (!\in_array($element, $allowedHeader)) {
					throw new Exception("Row with header '$element' is not allowed in table but found");
				}
			}
		}
	}

	/**
	 * Verify that the tableNode contains expected rows
	 *
	 * @param TableNode $table
	 * @param array $requiredRows
	 * @param array $allowedRows
	 *
	 * @return void
	 * @throws Exception
	 */
	public function verifyTableNodeRows(TableNode $table, array $requiredRows = [], array $allowedRows = []): void {
		if (\count($table->getRows()) < 1) {
			throw new Exception("Table should have at least one row.");
		}
		$tableHeaders = $table->getColumn(0);
		$allowedRows = \array_unique(\array_merge($requiredRows, $allowedRows));
		if ($requiredRows != []) {
			foreach ($requiredRows as $element) {
				if (!\in_array($element, $tableHeaders)) {
					throw new Exception("Row with name '$element' expected to be in table but not found");
				}
			}
		}

		if ($allowedRows != []) {
			foreach ($tableHeaders as $element) {
				if (!\in_array($element, $allowedRows)) {
					throw new Exception("Row with name '$element' is not allowed in table but found");
				}
			}
		}
	}

	/**
	 * Verify that the tableNode contains expected number of columns
	 *
	 * @param TableNode $table
	 * @param int $count
	 *
	 * @return void
	 * @throws Exception
	 */
	public function verifyTableNodeColumnsCount(TableNode $table, int $count): void {
		if (\count($table->getRows()) < 1) {
			throw new Exception("Table should have at least one row.");
		}
		$rowCount = \count($table->getRows()[0]);
		if ($count !== $rowCount) {
			throw new Exception("Table expected to have $count rows but found $rowCount");
		}
	}

	/**
	 * @param string $method http request method
	 * @param string $property property in form d:getetag
	 *                         if property is `doesnotmatter` body is also set `doesnotmatter`
	 *
	 * @return string
	 */
	public function getBodyForOCSRequest(string $method, string $property): ?string {
		$body = null;
		if ($method === 'PROPFIND') {
			$body = '<?xml version="1.0"?><d:propfind  xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns"><d:prop><'
			. $property . '/></d:prop></d:propfind>';
		} elseif ($method === 'LOCK') {
			$body = "<?xml version='1.0' encoding='UTF-8'?><d:lockinfo xmlns:d='DAV:'> <d:lockscope><"
			. $property . " /></d:lockscope></d:lockinfo>";
		} elseif ($method === 'PROPPATCH') {
			if ($property === 'favorite') {
				$property = '<oc:favorite xmlns:oc="http://owncloud.org/ns">1</oc:favorite>';
			}
			$body = '<?xml version="1.0"?><d:propertyupdate xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns"><d:set>'
			. "<d:prop>" . $property . '</d:prop></d:set></d:propertyupdate>';
		}
		if ($property === '') {
			$body = '';
		}
		return $body;
	}

	/**
	 * The method returns groupId
	 *
	 * @param string $groupName
	 *
	 * @return string
	 * @throws Exception|GuzzleException
	 */
	public function getGroupIdByGroupName(string $groupName): string {
		$response = GraphHelper::getGroup(
			$this->getBaseUrl(),
			$this->getStepLineRef(),
			$this->getAdminUsername(),
			$this->getAdminPassword(),
			$groupName
		);
		$data = $this->getJsonDecodedResponse($response);
		if (isset($data["id"])) {
			return $data["id"];
		} else {
			throw new Exception(__METHOD__ . " accounts-list is empty");
		}
	}

	/**
	 *
	 * @AfterSuite
	 *
	 * @return void
	 * @throws Exception
	 */
	public static function clearScenarioLog(): void {
		if (\file_exists(HttpLogger::getScenarioLogPath())) {
			\unlink(HttpLogger::getScenarioLogPath());
		}
	}

	/**
	 * Log request and response logs if scenario fails
	 *
	 * @AfterScenario
	 *
	 * @param AfterScenarioScope $scope
	 *
	 * @return void
	 * @throws Exception
	 */
	public static function checkScenario(AfterScenarioScope $scope): void {
		if (($scope->getTestResult()->getResultCode() !== 0)
			&& (!self::isExpectedToFail(self::getScenarioLine($scope)))
		) {
			$logs = \file_get_contents(HttpLogger::getScenarioLogPath());
			// add new lines
			$logs = \rtrim($logs, "\n") . "\n\n\n";
			HttpLogger::writeLog(HttpLogger::getFailedLogPath(), $logs);
		}
	}

	/**
	 * @param BeforeScenarioScope|AfterScenarioScope $scope
	 *
	 * @return string
	 */
	public static function getScenarioLine($scope): string {
		$feature = $scope->getFeature()->getFile();
		$feature = \explode('/', $feature);
		$feature = \array_slice($feature, -2);
		$feature = \implode('/', $feature);
		$scenarioLine = $scope->getScenario()->getLine();
		// Example: apiGraph/createUser.feature:24
		return $feature . ':' . $scenarioLine;
	}

	/**
	 * @param string $scenarioLine
	 *
	 * @return bool
	 */
	public static function isExpectedToFail(string $scenarioLine): bool {
		$expectedFailFile = \getenv('EXPECTED_FAILURES_FILE');
		if (!$expectedFailFile) {
			$expectedFailFile = __DIR__ . '/../expected-failures-localAPI-on-decomposed-storage.md';
			if (\strpos($scenarioLine, "coreApi") === 0) {
				$expectedFailFile = __DIR__ . '/../expected-failures-API-on-decomposed-storage.md';
			}
		}

		$reader = \fopen($expectedFailFile, 'r');
		if ($reader) {
			while (($line = \fgets($reader)) !== false) {
				if (\strpos($line, $scenarioLine) !== false) {
					\fclose($reader);
					return true;
				}
			}
			\fclose($reader);
		}
		return false;
	}
}
