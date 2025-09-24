package utils

ALLOWED_RESOURCE_EXTENSIONS := [
    ".apk", ".avi", ".bat", ".bmp", ".css", ".csv", ".doc", ".docm", ".docx",
    ".docxf", ".dotx", ".eml", ".epub", ".htm", ".html", ".ipa", ".jar", ".java",
    ".jpg", ".js", ".json", ".mp3", ".mp4", ".msg", ".odp", ".ods", ".odt", ".oform",
    ".ots", ".ott", ".pdf", ".php", ".png", ".potm", ".potx", ".ppsm", ".ppsx", ".ppt",
    ".pptm", ".pptx", ".py", ".rtf", ".sb3", ".sprite3", ".sql", ".svg", ".tif", ".tiff",
    ".txt", ".xls", ".xlsm", ".xlsx", ".xltm", ".xltx", ".xml", ".zip", ".md"
]

is_extension_allowed(identifier) {
     extension := ALLOWED_RESOURCE_EXTENSIONS[_]
     endswith(identifier, extension)
}

is_mimetype_allowed(mimetype) {
     extensions := opencloud.mimetype.extensions(mimetype)
     extension := extensions[_]
     is_extension_allowed(extension)
}

