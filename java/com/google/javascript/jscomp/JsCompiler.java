/*
 * Copyright 2016 The Closure Rules Authors. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.google.javascript.jscomp;

import static com.google.common.base.Verify.verifyNotNull;
import static java.nio.charset.StandardCharsets.UTF_8;

import com.google.common.collect.HashMultimap;
import com.google.common.collect.Iterators;
import com.google.common.collect.Multimap;
import com.google.common.collect.PeekingIterator;
import io.bazel.rules.closure.BuildInfo.ClosureJsLibrary;
import io.bazel.rules.closure.worker.CommandLineProgram;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import javax.inject.Inject;

/**
 * Closure Rules runner for Closure Compiler.
 *
 * <p>This program is a thin layer on top of the standard Closure Compiler CLI which is configured
 * with a superset of its flags. The following features have been added:
 *
 * <ol>
 * <li>Friendlier error messages
 * <li>All errors enabled by default
 * <li>Entirely new system for suppressing errors
 * <li>Flags for efficiently unit testing error messages from build rules
 * </ol>
 *
 * <p>The most important feature of this wrapper is that it overrides the error formatter to insert
 * helpful instructions on the various ways they can be suppressed. This is important because the
 * diagnostic codes associated with error messages are not obvious. See {@link JsCompilerWarnings}
 * for more information.
 *
 * <p>Now that instructions on how to suppress warnings are plain and obvious, we needn't show any
 * compunction in enabling the full suite of compiler warnings and treating them as errors. Every
 * single check the Closure Compiler is capable of performing is enabled by this wrapper, including
 * ones like reportUnknownTypes, which even {@code --jscomp_error=*} won't enable. Exceptions are
 * made for errors that are known to be noisy or problematic. See {@link Diagnostics} for more
 * information.
 *
 * <p>This compilation strategy optimized for greenfield projects. It makes the benefits of the
 * Closure Compiler significantly more accessible to the uninitiated. However this wrapper is also
 * able to support legacy codebases through awareness of the Bazel build graph. It reads the pbtxt
 * files generated by all closure_js_library rules in the transitive closure. It does this not only
 * to determine which errors should be suppressed in which files, but to also determine which
 * sources originated in legacy rules and should therefore be treated with the greater leniency.
 */
public final class JsCompiler implements CommandLineProgram {

  private static final byte[] EMPTY_BYTE_ARRAY = new byte[0];

  @Inject
  JsCompiler() {}

  @Override
  public Integer apply(Iterable<String> args) {
    try {
      return run(args);
    } catch (IOException e) {
      throw new RuntimeException(e);
    }
  }

  private int run(Iterable<String> args) throws IOException {
    // Our flags, which we won't pass along to the compiler.
    List<ClosureJsLibrary> infos = new ArrayList<>();
    List<String> roots = new ArrayList<>();
    Set<DiagnosticType> globalSuppressions = new HashSet<>();
    Path outputErrors = null;
    boolean expectFailure = false;
    boolean expectWarnings = false;
    boolean exportTestFunctions = false;
    boolean checksOnly = false;
    boolean disablePropertyRenaming = false;
    boolean devBuild = false;

    // Compiler flags we want to read.
    Path jsOutputFile = null;
    Path createSourceMap = null;

    // Parse flags in an ad-hoc manner.
    List<String> passThroughArgs = new ArrayList<>(1024);
    PeekingIterator<String> iargs = Iterators.peekingIterator(args.iterator());
    while (iargs.hasNext()) {
      String arg = iargs.next();
      switch (arg) {
        case "--info":
          infos.add(JsCheckerHelper.loadClosureJsLibraryInfo(Paths.get(iargs.next())));
          continue;
        case "--output_errors":
          outputErrors = Paths.get(iargs.next());
          continue;
        case "--suppress":
          globalSuppressions.addAll(Diagnostics.getDiagnosticTypesForSuppressCode(iargs.next()));
          continue;
        case "--expect_failure":
          expectFailure = true;
          continue;
        case "--expect_warnings":
          expectWarnings = true;
          continue;
        case "--export_test_functions":
          // TODO(jart): Remove this when it's added to open source Closure Compiler.
          exportTestFunctions = true;
          continue;
        case "--js_module_root":
          roots.add(iargs.peek());
          break;
        case "--js_output_file":
          jsOutputFile = Paths.get(iargs.peek());
          break;
        case "--checks_only":
          checksOnly = true;
          break;
        case "--create_source_map":
          createSourceMap = Paths.get(iargs.peek());
          break;
        case "--disable_property_renaming":
          disablePropertyRenaming = true;
          continue;
        case "--experimental_dev_build":
          devBuild = true;
          continue;
        default:
          break;
      }
      passThroughArgs.add(arg);
    }

    // Keep track of modules defined *only* in rules that don't have the suppress attribute,
    // e.g. js_library(). We'll make the compiler much more lax for these sources.
    Set<String> legacyModules = new HashSet<>();
    for (ClosureJsLibrary info : infos) {
      if (info.getLegacy()) {
        legacyModules.addAll(info.getModuleList());
      }
    }

    // We want to be able to turn module names back into labels.
    Map<String, String> labels = new HashMap<>();
    // We also need the set of all (module, suppress) combinations propagated from library rules.
    Multimap<String, DiagnosticType> suppressions = HashMultimap.create();
    for (ClosureJsLibrary info : infos) {
      if (info.getLegacy()) {
        continue;
      }
      legacyModules.removeAll(info.getModuleList());
      for (String module : info.getModuleList()) {
        labels.put(module, info.getLabel());
        for (String suppress : info.getSuppressList()) {
          suppressions.put(module,
              verifyNotNull(Diagnostics.DIAGNOSTIC_TYPES.get(suppress),
                  "Bad DiagnosticType from closure_js_library: %s", suppress));
        }
      }
    }

    // Run the compiler, capturing error messages.
    boolean failed = false;
    Compiler compiler = new Compiler();
    JsCheckerErrorFormatter errorFormatter = new JsCheckerErrorFormatter(compiler, roots, labels);
    errorFormatter.setColorize(true);
    JsCheckerErrorManager errorManager = new JsCheckerErrorManager(errorFormatter);
    compiler.setErrorManager(errorManager);
    JsCompilerWarnings warnings =
        new JsCompilerWarnings(roots, legacyModules, suppressions, globalSuppressions);
    JsCompilerRunner runner =
        new JsCompilerRunner(
            passThroughArgs,
            compiler,
            exportTestFunctions,
            warnings,
            disablePropertyRenaming,
            devBuild);
    if (runner.shouldRunCompiler()) {
      failed |= runner.go() != 0;
    }
    failed |= runner.hasErrors();

    // Output error messages based on diagnostic settings.
    if (!expectFailure && !expectWarnings) {
      for (String line : errorManager.stderr) {
        System.err.println(line);
      }
      System.err.flush();
    }
    if (outputErrors != null) {
      Files.write(outputErrors, errorManager.stderr, UTF_8);
    }
    if ((failed && expectFailure) || checksOnly) {
      // If we don't return nonzero, Bazel expects us to create every output file.
      if (jsOutputFile != null) {
        Files.write(jsOutputFile, EMPTY_BYTE_ARRAY);
      }
    }

    // Make sure a source map is always created since Bazel expect that but JsCompiler
    // may not emit sometimes (e.g compiler_level=BUNLDE)
    if (createSourceMap != null && !Files.exists(createSourceMap)) {
      Files.write(createSourceMap, EMPTY_BYTE_ARRAY);
    }

    if (!failed && expectFailure) {
      System.err.println("ERROR: Expected failure but didn't fail.");
    }
    return failed == expectFailure ? 0 : 1;
  }
}
