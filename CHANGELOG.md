# CHANGELOG

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
