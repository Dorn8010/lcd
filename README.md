# lcd - Library-based Change Directory
Usage: lcd [options] <directory_name_or_fragment>

A system wide scan library based cd (Library Change Directory) with a small, fast and simple application. On first start, a library of the directory (can be changed via parameters) tree starting from the home directory will be created. A bash and GO version is available for Linux and MacOS.

## Description</br>
  Fast directory navigation using a cached directory tree (~/.lcd-tree.txt).</br>
  First run will index your home directory automatically, with lcd --rescan you can force the rescanning of the directories.</br>
  Typing exit in the CLI brings you back to the old directory.</br>
  (C) 2025 by Alexander Dorn, MIT license</br>
</br>
## Options
  --verbose, -v      Show detailed logs during operation</br>
  --print            Print the found path to stdout (do not cd)</br>
  --copy             Copy the found path to system clipboard</br>
  --rescan           Force a rescan of the filesystem</br>
  --newbasedir <dir> Set a new root directory for scanning (implies --rescan)</br>
  --version          Show version info</br>
  --help, -h         Show this help message</br>
</br>
</br>
## Examples
Change to the directory from anywhere: lcd myprojectfolder</br>
Change to hidden directory from anywhere: lcd .myhidden</br>
Change to first folder with this string found: lcd proj</br>
Print directory path : lcd --print myprojectfolder</br>
Copy path to clipboard : lcd --copy myprojectfolder</br>
Rescan all folders from the root directory : lcd --rescan --newbasedir /</br>
</br>
## Search Logic
  1. Searches for an Exact Match (case-insensitive) of the directory name.</br>
  2. If not found, searches for a Partial Match.</br>
