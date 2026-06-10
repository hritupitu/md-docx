// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "md2docx",
    platforms: [.macOS(.v13)],
    targets: [
        .executableTarget(
            name: "md2docx",
            path: "Sources/md2docx"
        )
    ]
)
