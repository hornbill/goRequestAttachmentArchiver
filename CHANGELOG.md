# CHANGELOG

## 1.4.0 (February 10th, 2026)

Features:

- Allow for processing multiple specific requests in CSV-ish file (using -requestList command line parameter; provide file name of file containing list of Request IDs - one per line)


## 1.3.0 (November 4th, 2024)

Features:

- Addition of filtering by Status (Statuses-Array of strings within configuration file (eg: ["status.cancelled","status.closed"] to only syphon off from cancelled/closed requests)). Please note NOT giving ANY Statuses will (still) result in ALL applicable requests.

## 1.2.2 (October 30th, 2024)

Fixes:

- fix to code implemented in v1.2.1

## 1.2.1 (September 25th, 2024)

Fixes:

- Modification to account for change of naming convention within API - functionality remains the same


## 1.2.0 (September 6th, 2023)

Features:

- Ability to skip the archiving of the attachments in the .zip file - i.e. file REMOVAL WITHOUT ARCHIVING (use -DoNotArchiveFiles command line flag).

## 1.1.0 (September 6th, 2023)

Features:

- Ability to insert diary entry to request, listing which files have been archived (use -updateRequest command line flag).
- Addition of filtering by Service ID (Services-Array of integers within configuration file). Please note, that IF one decides to use this, that NOT giving ANY Service IDs will (still) result in ALL applicable requests. In other words, if one decides to single out a single service for different timings, then you will likely be using a second configuration file listing ALL the OTHER services.

## 1.0.0 (August 13th, 2021)

Features:

- Initial Release
