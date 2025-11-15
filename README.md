# zus-tools
A collection of operational tools for Züs infrastructure, including monitoring, capacity automation, and maintenance utilities.

## 🌱 Development Branch

The **development** branch is the primary integration branch for all ongoing work in the `zus-tools` repository.  
All new features, bug fixes, and improvements should be merged into this branch before they reach `main`.

### Purpose of the Development Branch

- Acts as the collaboration hub for active development.  
- Allows multiple contributors to work on different tools and scripts without affecting the stable `main` branch.  
- Ensures that new changes are tested, reviewed, and validated before being promoted to production-ready code.  
- Helps maintain a clean and stable `main` branch that always reflects the latest reliable version of the repository.

### Recommended Workflow

1. Create a feature branch from `development`:
   ```bash
   git checkout development
   git pull
   git checkout -b feature/my-new-tool-or-feature
   ```
2. Commit and push your changes.

3. Open a ***Pull Request into development***.

4. After review and validation, changes will be merged into development.

5. Periodically, stable updates from development will be merged into main.

## Why Not Commit Directly to main?

-   main should remain stable, deployable, and clean.

-   Direct commits risk introducing breaking changes or incomplete features.

-   The development branch provides a safe environment for iterative development.