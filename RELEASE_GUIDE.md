# Creating a GitHub Release for OpenPrompt

This guide explains how to create a new release for OpenPrompt, which will automatically build binaries for Windows, macOS, and Linux.

## Prerequisites

- Push access to the GitHub repository
- Git installed on your local machine

## Steps to Create a Release

1. **Ensure your code is ready for release**
   - All features are implemented and tested
   - README and documentation are up to date

2. **Create and push a new tag**
   ```bash
   # Make sure you're on the main branch with the latest changes
   git checkout main
   git pull

   # Create a new tag (replace X.Y.Z with the version number, e.g., 1.0.0)
   git tag vX.Y.Z

   # Push the tag to GitHub
   git push origin vX.Y.Z
   ```

3. **Wait for the GitHub Actions workflow to complete**
   - Go to the "Actions" tab in your GitHub repository
   - You should see a workflow named "Release OpenPrompt" running
   - Wait for it to complete (this may take a few minutes)

4. **Verify and edit the release**
   - Go to the "Releases" tab in your GitHub repository
   - You should see a new draft release created with your tag
   - Click "Edit" to add release notes and description
   - Include information about:
     - New features
     - Bug fixes
     - Breaking changes
     - Installation instructions

5. **Publish the release**
   - Once you've added all the necessary information, click "Publish release"
   - Your release is now live and users can download the binaries

## Versioning Guidelines

Follow semantic versioning (SemVer) for your releases:
- **MAJOR** version (X.0.0) for incompatible API changes
- **MINOR** version (0.X.0) for new functionality in a backward compatible manner
- **PATCH** version (0.0.X) for backward compatible bug fixes

## Troubleshooting

If the GitHub Actions workflow fails:
1. Go to the "Actions" tab and click on the failed workflow
2. Examine the logs to identify the issue
3. Fix the issue in your code
4. Delete the tag, make your changes, and create a new tag:
   ```bash
   # Delete the tag locally
   git tag -d vX.Y.Z
   
   # Delete the tag on GitHub
   git push --delete origin vX.Y.Z
   
   # After fixing issues, create and push a new tag
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```
