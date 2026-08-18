<?php declare(strict_types=1);
/**
 * @author Sajan Gurung <sajan@jankaritech.com>
 * @copyright Copyright (c) 2026 Sajan Gurung sajan@jankaritech.com
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

/**
 * Most of the code is copied from https://github.com/ankitpokhrel/tus-php/blob/main/src/Tus/Client.php
 */

namespace TestHelpers;

use GuzzleHttp\Client as GuzzleClient;
use GuzzleHttp\Exception\ClientException;
use GuzzleHttp\Exception\GuzzleException;
use GuzzleHttp\Exception\ConnectException;
use Symfony\Component\HttpFoundation\Response as HttpResponse;

/**
 * TUS client for uploading files using the TUS protocol.
 */
class TUSClient {
	/**
	 * @const string Tus protocol version.
	 */
	public const TUS_PROTOCOL_VERSION = '1.0.0';

	/**
	 * @const string Header Content Type
	 */
	protected const HEADER_CONTENT_TYPE = 'application/offset+octet-stream';

	/**
	 * @const Input stream
	 */
	public const INPUT_STREAM = 'php://input';

	/**
	 * @const Read binary mode
	 */
	public const READ_BINARY = 'rb';

	/**
	 * @var GuzzleClient
	 */
	protected $client;

	/**
	 * @var string
	 */
	protected $apiPath = '/files';

	/**
	 * @var string
	 */
	protected $filePath;

	/**
	 * @var int
	 */
	protected $fileSize = 0;

	/**
	 * @var string
	 */
	protected $fileName;

	/**
	 * @var string
	 */
	protected $url;

	/**
	 * @var string
	 */
	protected $checksum;

	/**
	 * @var int
	 */
	protected $partialOffset = -1;

	/**
	 * @var string
	 */
	protected $checksumAlgorithm = 'sha256';

	/**
	 * @var array
	 */
	protected $metadata = [];

	/**
	 * @var array
	 */
	protected $headers = [];

	/**
	 * Client constructor.
	 *
	 * @param string $baseUri
	 * @param array  $options
	 *
	 * @throws \ReflectionException
	 */
	public function __construct(string $baseUri, array $options = []) {
		$this->headers      = $options['headers'] ?? [];
		$options['headers'] = [
			'Tus-Resumable' => self::TUS_PROTOCOL_VERSION,
		] + ($this->headers);

		$this->client = new GuzzleClient(
			['base_uri' => $baseUri] + $options
		);
	}

	/**
	 * Get guzzle client.
	 *
	 * @return GuzzleClient
	 */
	public function getClient(): GuzzleClient {
		return $this->client;
	}

	/**
	 * Get file size.
	 *
	 * @return int
	 */
	public function getFileSize(): int {
		return $this->fileSize;
	}

	/**
	 * Get file path.
	 *
	 * @return string|null
	 */
	public function getFilePath(): ?string {
		return $this->filePath;
	}

	/**
	 * Get url.
	 *
	 * @return string|null
	 */
	public function getUrl(): ?string {
		return $this->url;
	}

	/**
	 * Set API path.
	 *
	 * @param string $path
	 *
	 * @return self
	 */
	public function setApiPath(string $path): self {
		$this->apiPath = $path;

		return $this;
	}

	/**
	 * Set checksum algorithm.
	 *
	 * @param string $algorithm
	 *
	 * @return Client
	 */
	public function setChecksumAlgorithm(string $algorithm): self {
		$this->checksumAlgorithm = $algorithm;

		return $this;
	}

	/**
	 * Get checksum algorithm.
	 *
	 * @return string
	 */
	public function getChecksumAlgorithm(): string {
		return $this->checksumAlgorithm;
	}

	/**
	 * Set checksum.
	 *
	 * @param string $checksum
	 *
	 * @return Client
	 */
	public function setChecksum(string $checksum): self {
		$this->checksum = $checksum;

		return $this;
	}

	/**
	 * Get checksum.
	 *
	 * @return string
	 */
	public function getChecksum(): string {
		if (empty($this->checksum)) {
			$this->setChecksum(hash_file($this->getChecksumAlgorithm(), $this->getFilePath()));
		}

		return $this->checksum;
	}

	/**
	 * Get upload checksum header.
	 *
	 * @return string
	 */
	protected function getUploadChecksumHeader(): string {
		return $this->getChecksumAlgorithm() . ' ' . base64_encode($this->getChecksum());
	}

	/**
	 * Set metadata.
	 *
	 * @param array $items
	 *
	 * @return Client
	 */
	public function setMetadata(array $items): self {
		$items = array_map('base64_encode', $items);

		$this->metadata = $items;

		return $this;
	}

	/**
	 * Get metadata.
	 *
	 * @return array
	 */
	public function getMetadata(): array {
		return $this->metadata;
	}

	/**
	 * Add metadata.
	 *
	 * @param string $key
	 * @param string $value
	 *
	 * @return Client
	 */
	public function addMetadata(string $key, string $value): self {
		$this->metadata[$key] = base64_encode($value);

		return $this;
	}

	/**
	 * Get metadata for Upload-Metadata header.
	 *
	 * @return string
	 */
	protected function getUploadMetadataHeader(): string {
		$metadata = [];

		foreach ($this->getMetadata() as $key => $value) {
			$metadata[] = "{$key} {$value}";
		}

		return implode(',', $metadata);
	}

	/**
	 * Set file properties.
	 *
	 * @param string      $file File path.
	 * @param string|null $name File name.
	 *
	 * @return Client
	 */
	public function file(string $file, string $name = null): self {
		$this->filePath = $file;

		if (! file_exists($file) || ! is_readable($file)) {
			throw new FileException('Cannot read file: ' . $file);
		}

		$this->fileName = $name ?? basename($this->filePath);
		$this->fileSize = filesize($file);

		$this->addMetadata('filename', $this->fileName);

		return $this;
	}

	/**
	 * Handle client exception during patch request.
	 *
	 * @param ClientException $e
	 *
	 * @return \Exception
	 */
	protected function handleClientException(ClientException $e) {
		$response   = $e->getResponse();
		$statusCode = $response !== null ? $response->getStatusCode() : HttpResponse::HTTP_INTERNAL_SERVER_ERROR;

		if ($statusCode === HttpResponse::HTTP_REQUESTED_RANGE_NOT_SATISFIABLE) {
			return new FileException('The uploaded file is corrupt.');
		}

		if ($statusCode === HttpResponse::HTTP_CONTINUE) {
			return new ConnectionException('Connection aborted by user.');
		}

		if ($statusCode === HttpResponse::HTTP_UNSUPPORTED_MEDIA_TYPE) {
			return new TusException('Unsupported media types.');
		}

		return new TusException((string) $response->getBody(), $statusCode);
	}

	/**
	 * Send HEAD request.
	 *
	 * @throws FileException
	 * @throws GuzzleException
	 *
	 * @return int
	 */
	protected function sendHeadRequest(): int {
		$response   = $this->getClient()->head($this->getUrl());
		$statusCode = $response->getStatusCode();

		if ($statusCode !== HttpResponse::HTTP_OK) {
			throw new FileException('File not found.');
		}

		return (int) current($response->getHeader('upload-offset'));
	}

	/**
	 * Send PATCH request.
	 *
	 * @param int $bytes
	 * @param int $offset
	 *
	 * @throws TusException
	 * @throws FileException
	 * @throws GuzzleException
	 * @throws ConnectionException
	 *
	 * @return int
	 */
	protected function sendPatchRequest(int $bytes, int $offset): int {
		$data    = $this->getData($offset, $bytes);
		$headers = $this->headers + [
			'Content-Type' => self::HEADER_CONTENT_TYPE,
			'Content-Length' => (string)\strlen($data),
			'Upload-Checksum' => $this->getUploadChecksumHeader(),
			'Upload-Offset' => $offset,
		];

		try {
			$response = $this->getClient()->patch(
				$this->getUrl(),
				[
					'body' => $data,
					'headers' => $headers,
				]
			);

			return (int) current($response->getHeader('upload-offset'));
		} catch (ClientException $e) {
			throw $this->handleClientException($e);
		} catch (ConnectException $e) {
			throw new ConnectionException("Couldn't connect to server.");
		}
	}

	/**
	 * Check if file to read exists.
	 *
	 * @param string $filePath
	 * @param string $mode
	 *
	 * @throws FileException
	 *
	 * @return bool
	 */
	public function exists(string $filePath, string $mode = self::READ_BINARY): bool {
		if ($filePath === self::INPUT_STREAM) {
			return true;
		}

		if ($mode === self::READ_BINARY && ! file_exists($filePath)) {
			throw new FileException('File not found.');
		}

		return true;
	}

	/**
	 * Get X bytes of data from file.
	 *
	 * @param int $offset
	 * @param int $bytes
	 *
	 * @return string
	 */
	protected function getData(int $offset, int $bytes): string {
		$filePath = $this->getFilePath();
		$mode = self::READ_BINARY;

		$this->exists($filePath, $mode);
		$handle = @fopen($filePath, $mode);

		if ($handle === false) {
			throw new FileException("Unable to open $filePath.");
		}

		$position = fseek($handle, $offset, SEEK_SET);
		if ($position === -1) {
			throw new FileException('Cannot move pointer to desired position.');
		}

		$data = fread($handle, $bytes);
		if ($data === false) {
			throw new FileException('Cannot read file.');
		}

		fclose($handle);
		return $data;
	}

	/**
	 * Upload file.
	 *
	 * @param int $bytes Bytes to upload
	 *
	 * @throws TusException
	 * @throws GuzzleException
	 * @throws ConnectionException
	 *
	 * @return int
	 */
	public function upload(int $bytes = -1): int {
		$bytes  = $bytes < 0 ? $this->getFileSize() : $bytes;
		$offset = $this->partialOffset < 0 ? 0 : $this->partialOffset;

		try {
			// Check if this upload exists with HEAD request.
			$offset = $this->sendHeadRequest();
		} catch (FileException | ClientException $e) {
			// Create a new upload.
			$this->url = $this->create();
		} catch (ConnectException $e) {
			throw new ConnectionException("Couldn't connect to server.");
		}

		// Now, resume upload with PATCH request.
		return $this->sendPatchRequest($bytes, $offset);
	}

	/**
	 * Create resource with POST request.
	 *
	 * @throws FileException
	 * @throws GuzzleException
	 *
	 * @return string
	 */
	public function create(): string {
		return $this->createWithUpload(0)['location'];
	}

	/**
	 * Create resource with POST request and upload data using the creation-with-upload extension.
	 *
	 * @see https://tus.io/protocols/resumable-upload.html#creation-with-upload
	 *
	 * @param int    $bytes -1 => all data; 0 => no data
	 *
	 * @throws GuzzleException
	 *
	 * @return array [
	 *     'location' => string,
	 *     'offset' => int
	 * ]
	 */
	public function createWithUpload(int $bytes = -1): array {
		$bytes = $bytes < 0 ? $this->fileSize : $bytes;

		$headers = $this->headers + [
			'Upload-Length' => (string)$this->fileSize,
			'Upload-Checksum' => $this->getUploadChecksumHeader(),
			'Upload-Metadata' => $this->getUploadMetadataHeader(),
		];

		$data = '';
		if ($bytes > 0) {
			$data = $this->getData(0, $bytes);

			$headers += [
				'Content-Type' => self::HEADER_CONTENT_TYPE,
				'Content-Length' => (string)\strlen($data),
			];
		}

		try {
			$response = $this->getClient()->post(
				$this->apiPath,
				[
					'body' => $data,
					'headers' => $headers,
				]
			);
		} catch (ClientException $e) {
			$response = $e->getResponse();
		}

		$statusCode = $response->getStatusCode();

		if ($statusCode !== HttpResponse::HTTP_CREATED) {
			throw new FileException('Unable to create resource.');
		}

		$uploadOffset   = $bytes > 0 ? current($response->getHeader('upload-offset')) : 0;
		$uploadLocation = current($response->getHeader('location'));

		return [
			'location' => $uploadLocation,
			'offset' => $uploadOffset,
		];
	}
}

/**
 * --------------------------------
 * Exceptions
 * --------------------------------
 */

/**
 * File exception class.
 */
class FileException extends \RuntimeException {
}

/**
 * Connection exception class.
 */
class ConnectionException extends \Exception {
}

/**
 * Tus exception class.
 */
class TusException extends \Exception {
}
