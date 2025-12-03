# Gemini Development Guidelines for `rubi`

## 1. Introduction

This document defines the development process and guidelines for how Gemini will contribute to the `rubi` project.

## 2. Development Policy

- **Source of Truth:** All development will be based on the definitions and requirements outlined in `REQUIREMENTS.md`.
- **Go Version:** To ensure broad accessibility for many contributors, we will use **Go version 1.21**. This allows for a stable and widely adopted development environment.
- **Command Automation:** Basic and safe command-line operations (e.g., `go fmt`, `go test`, `git add`) will be executed automatically to streamline the development process.

## 3. Development Cycle (Issue-Driven)

This project will follow an Issue-Driven Development workflow.

1.  **Issue Creation by Gemini:** Before starting development, based on the `REQUIREMENTS.md`, Gemini will create a GitHub Issue for the development plan.
2.  **Pull Request by Gemini:** Gemini will implement a feature or fix. When creating a Pull Request, it will be based on the corresponding Issue.
3.  **Code Review by You:** You will review the submitted Pull Request, providing feedback or approval.
4.  **Merge:** Once approved, you will merge the Pull Request into the main branch, which will automatically close the corresponding Issue.

This iterative process ensures clarity, traceability, and alignment with the project goals.
