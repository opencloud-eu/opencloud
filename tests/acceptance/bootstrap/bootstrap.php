<?php declare(strict_types=1);

/**
 * @author Phil Davis <phil@jankaritech.com>
 * @copyright Copyright (c) 2020 Phil Davis phil@jankaritech.com
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

use Composer\Autoload\ClassLoader;

$classLoader = new ClassLoader();

$classLoader->addPsr4("TestHelpers\\", __DIR__ . "/../TestHelpers", true);

$classLoader->register();

// Default number of times to retry where retries are useful
if (!\defined('STANDARD_RETRY_COUNT')) {
	\define('STANDARD_RETRY_COUNT', 10);
}
// Minimum number of times to retry where retries are useful
if (!\defined('MINIMUM_RETRY_COUNT')) {
	\define('MINIMUM_RETRY_COUNT', 2);
}

// Minimum number of times to retry where retries are useful
if (!\defined('HTTP_REQUEST_TIMEOUT')) {
	\define('HTTP_REQUEST_TIMEOUT', 60);
}

// The remote server-under-test might or might not happen to have this directory.
// If it does not exist, then the tests may end up creating it.
if (!\defined('ACCEPTANCE_TEST_DIR_ON_REMOTE_SERVER')) {
	\define('ACCEPTANCE_TEST_DIR_ON_REMOTE_SERVER', 'tests/acceptance');
}

// The following directory should NOT already exist on the remote server-under-test.
// Acceptance tests are free to do anything needed in this directory, and to
// delete it during or at the end of testing.
if (!\defined('TEMPORARY_STORAGE_DIR_ON_REMOTE_SERVER')) {
	\define('TEMPORARY_STORAGE_DIR_ON_REMOTE_SERVER', ACCEPTANCE_TEST_DIR_ON_REMOTE_SERVER . '/server_tmp');
}
