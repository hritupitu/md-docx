import SwiftUI
import AppKit
import UniformTypeIdentifiers

// MARK: - Model

struct FileEntry: Identifiable {
    let id = UUID()
    let url: URL
    var name: String { url.lastPathComponent }
    var state: EntryState = .ready
    var savedName: String? = nil
    var errorMsg: String? = nil

    enum EntryState { case ready, converting, done, failed }
}

// MARK: - Helpers

func findPandoc() -> String? {
    if let exe = Bundle.main.executablePath {
        let bundled = URL(fileURLWithPath: exe)
            .deletingLastPathComponent()
            .appendingPathComponent("pandoc").path
        if FileManager.default.fileExists(atPath: bundled) { return bundled }
    }
    for p in ["/opt/homebrew/bin/pandoc", "/usr/local/bin/pandoc", "/usr/bin/pandoc"] {
        if FileManager.default.fileExists(atPath: p) { return p }
    }
    let t = Process(); t.executableURL = URL(fileURLWithPath: "/usr/bin/which")
    t.arguments = ["pandoc"]
    let pipe = Pipe(); t.standardOutput = pipe
    try? t.run(); t.waitUntilExit()
    let out = String(data: pipe.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8)?
        .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    return out.isEmpty ? nil : out
}

func runSavePanel(suggestedName: String) async -> URL? {
    await withCheckedContinuation { continuation in
        DispatchQueue.main.async {
            let panel = NSSavePanel()
            panel.nameFieldStringValue = suggestedName
            if let t = UTType(filenameExtension: "docx") { panel.allowedContentTypes = [t] }
            panel.prompt = "Save"
            panel.begin { response in
                continuation.resume(returning: response == .OK ? panel.url : nil)
            }
        }
    }
}

// MARK: - Main View

struct ContentView: View {
    @State private var entries: [FileEntry] = []
    @State private var refDocURL: URL? = nil
    @State private var converting = false
    @State private var pandocVersion: String? = nil
    @State private var pandocPath: String? = nil

    private let accent   = Color(red: 0, green: 212/255, blue: 1)
    private let bgColor  = Color(red: 0.051, green: 0.051, blue: 0.051)
    private let surface  = Color(white: 0.086)
    private let border   = Color(white: 0.165)

    var body: some View {
        ZStack {
            bgColor.ignoresSafeArea()
            ScrollView {
                VStack(spacing: 16) {
                    headerSection
                    pickZone
                    refSection
                    if !entries.isEmpty { fileList }
                    if !entries.isEmpty { actionBar }
                    Spacer(minLength: 40)
                }
                .padding(.horizontal, 20)
                .padding(.vertical, 32)
            }
        }
        .frame(width: 560, height: 700)
        .onAppear(perform: checkPandoc)
    }

    // MARK: - Sections

    private var headerSection: some View {
        VStack(spacing: 6) {
            Text("md2docx")
                .font(.system(size: 26, weight: .bold))
                .foregroundColor(accent)
            Text("Convert Markdown files to Word documents — locally, no cloud.")
                .font(.system(size: 13))
                .foregroundColor(.secondary)

            if let ver = pandocVersion {
                Text("pandoc \(ver)")
                    .font(.system(size: 11))
                    .foregroundColor(.green)
                    .padding(.horizontal, 10).padding(.vertical, 3)
                    .overlay(Capsule().stroke(Color.green, lineWidth: 1))
            } else {
                Text("pandoc not found — install: brew install pandoc")
                    .font(.system(size: 11))
                    .foregroundColor(.red)
                    .padding(.horizontal, 10).padding(.vertical, 3)
                    .overlay(Capsule().stroke(Color.red, lineWidth: 1))
            }
        }
    }

    private var pickZone: some View {
        Button { pickMDFiles() } label: {
            VStack(spacing: 10) {
                Text("📝").font(.system(size: 36))
                Text("Click to choose .md files")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundColor(.primary)
                Text("select one or more Markdown files")
                    .font(.system(size: 12))
                    .foregroundColor(.secondary)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 44)
            .background(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(border, lineWidth: 2)
            )
        }
        .buttonStyle(.plain)
        .disabled(converting)
    }

    private var refSection: some View {
        HStack(spacing: 10) {
            Text("📎")
            if let ref = refDocURL {
                Text(ref.lastPathComponent)
                    .font(.system(size: 13))
                    .lineLimit(1)
                Spacer()
                Button("✕ remove") { refDocURL = nil }
                    .font(.system(size: 12))
                    .foregroundColor(.red)
                    .buttonStyle(.plain)
            } else {
                Text("Optional: attach a reference .docx for Word styles & fonts")
                    .font(.system(size: 13))
                    .foregroundColor(.secondary)
                Spacer()
                Button("Choose…") { pickRefDoc() }
                    .font(.system(size: 12))
                    .foregroundColor(accent)
                    .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, 14).padding(.vertical, 10)
        .background(RoundedRectangle(cornerRadius: 8).stroke(border, lineWidth: 1))
        .disabled(converting)
    }

    private var fileList: some View {
        VStack(spacing: 8) {
            ForEach(entries.indices, id: \.self) { i in
                FileRow(entry: entries[i], accent: accent, border: border) {
                    if !converting { entries.remove(at: i) }
                }
            }
        }
    }

    private var actionBar: some View {
        HStack(spacing: 10) {
            Button(action: startConvert) {
                Text("Convert to DOCX")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundColor(.black)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 13)
                    .background(converting || pandocPath == nil ? accent.opacity(0.3) : accent)
                    .cornerRadius(8)
            }
            .buttonStyle(.plain)
            .disabled(converting || pandocPath == nil)

            Button {
                if !converting { entries = [] }
            } label: {
                Text("Clear")
                    .font(.system(size: 14))
                    .foregroundColor(.secondary)
                    .padding(.horizontal, 16).padding(.vertical, 13)
                    .background(RoundedRectangle(cornerRadius: 8).stroke(border, lineWidth: 1))
            }
            .buttonStyle(.plain)
            .disabled(converting)
        }
    }

    // MARK: - Logic

    private func pickMDFiles() {
        let panel = NSOpenPanel()
        panel.allowsMultipleSelection = true
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.message = "Choose Markdown files to convert"
        panel.prompt = "Select"
        panel.begin { response in
            guard response == .OK else { return }
            let existing = Set(self.entries.map(\.url))
            let valid = panel.urls.filter {
                let ext = $0.pathExtension.lowercased()
                return (ext == "md" || ext == "markdown") && !existing.contains($0)
            }
            self.entries.append(contentsOf: valid.map { FileEntry(url: $0) })
        }
    }

    private func pickRefDoc() {
        let panel = NSOpenPanel()
        panel.allowsMultipleSelection = false
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.message = "Choose a reference .docx file"
        panel.prompt = "Select"
        if let t = UTType(filenameExtension: "docx") { panel.allowedContentTypes = [t] }
        panel.begin { response in
            guard response == .OK else { return }
            self.refDocURL = panel.url
        }
    }

    private func checkPandoc() {
        Task.detached {
            guard let bin = findPandoc() else { return }
            let task = Process()
            task.executableURL = URL(fileURLWithPath: bin)
            task.arguments = ["--version"]
            let pipe = Pipe(); task.standardOutput = pipe
            try? task.run(); task.waitUntilExit()
            let raw = String(data: pipe.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
            let ver = raw.split(separator: " ").dropFirst().first.map(String.init) ?? ""
            await MainActor.run {
                pandocPath = bin
                pandocVersion = ver
            }
        }
    }

    private func startConvert() {
        guard !converting, !entries.isEmpty, let bin = pandocPath else { return }
        converting = true

        Task {
            for i in entries.indices {
                await MainActor.run { entries[i].state = .converting }

                let srcURL = entries[i].url
                let base   = srcURL.deletingPathExtension().lastPathComponent
                let tmpDir = FileManager.default.temporaryDirectory
                    .appendingPathComponent(UUID().uuidString)
                try? FileManager.default.createDirectory(at: tmpDir, withIntermediateDirectories: true)
                let outURL = tmpDir.appendingPathComponent(base + ".docx")

                var args = [srcURL.path, "-o", outURL.path, "--from=markdown", "--to=docx"]
                if let ref = refDocURL { args.append("--reference-doc=\(ref.path)") }

                let task = Process()
                task.executableURL = URL(fileURLWithPath: bin)
                task.arguments = args
                let errPipe = Pipe(); task.standardError = errPipe

                do {
                    try task.run()
                    task.waitUntilExit()

                    if task.terminationStatus == 0,
                       FileManager.default.fileExists(atPath: outURL.path) {
                        if let dst = await runSavePanel(suggestedName: base + ".docx") {
                            try? FileManager.default.removeItem(at: dst)
                            try FileManager.default.copyItem(at: outURL, to: dst)
                            await MainActor.run {
                                entries[i].state     = .done
                                entries[i].savedName = dst.lastPathComponent
                            }
                        } else {
                            // user cancelled — mark done without a save location
                            await MainActor.run { entries[i].state = .done }
                        }
                    } else {
                        let msg = String(
                            data: errPipe.fileHandleForReading.readDataToEndOfFile(),
                            encoding: .utf8
                        )?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "pandoc failed"
                        await MainActor.run {
                            entries[i].state    = .failed
                            entries[i].errorMsg = msg
                        }
                    }
                } catch {
                    await MainActor.run {
                        entries[i].state    = .failed
                        entries[i].errorMsg = error.localizedDescription
                    }
                }

                try? FileManager.default.removeItem(at: tmpDir)
            }

            await MainActor.run { converting = false }
        }
    }
}

// MARK: - File Row

struct FileRow: View {
    let entry: FileEntry
    let accent: Color
    let border: Color
    let onRemove: () -> Void

    private var rowBorder: Color {
        switch entry.state {
        case .ready:      return border
        case .converting: return accent
        case .done:       return .green
        case .failed:     return .red
        }
    }

    var body: some View {
        HStack(spacing: 12) {
            Text("📝").font(.system(size: 18))

            VStack(alignment: .leading, spacing: 2) {
                Text(entry.name)
                    .font(.system(size: 13))
                    .lineLimit(1)
                if let saved = entry.savedName {
                    Text("saved as \(saved)")
                        .font(.system(size: 11))
                        .foregroundColor(.green)
                } else if let err = entry.errorMsg {
                    Text(err)
                        .font(.system(size: 11))
                        .foregroundColor(.red)
                        .lineLimit(2)
                }
            }

            Spacer()

            switch entry.state {
            case .ready:
                Text("ready").font(.system(size: 13)).foregroundColor(.secondary)
                Button("✕", action: onRemove)
                    .buttonStyle(.plain).foregroundColor(.secondary)
            case .converting:
                ProgressView().scaleEffect(0.75)
            case .done:
                Text("✔").foregroundColor(.green).font(.system(size: 14, weight: .semibold))
            case .failed:
                Text("✗").foregroundColor(.red).font(.system(size: 14, weight: .semibold))
            }
        }
        .padding(.horizontal, 14).padding(.vertical, 10)
        .background(RoundedRectangle(cornerRadius: 8).stroke(rowBorder, lineWidth: 1))
        .animation(.easeInOut(duration: 0.2), value: entry.state)
    }
}
