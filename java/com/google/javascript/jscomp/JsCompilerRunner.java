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

import com.google.common.collect.Iterables;
import com.google.javascript.jscomp.deps.ModuleLoader;
import java.io.IOException;

final class JsCompilerRunner extends CommandLineRunner {

  private final Compiler compiler;
  private final boolean exportTestFunctions;
  private final WarningsGuard warnings;
  private final boolean disablePropertyRenaming;
  private final boolean devBuild;

  JsCompilerRunner(
      Iterable<String> args,
      Compiler compiler,
      boolean exportTestFunctions,
      WarningsGuard warnings,
      boolean disablePropertyRenaming,
      boolean devBuild) {
    super(Iterables.toArray(args, String.class));
    this.compiler = compiler;
    this.exportTestFunctions = exportTestFunctions;
    this.warnings = warnings;
    this.disablePropertyRenaming = disablePropertyRenaming;
    this.devBuild = devBuild;
  }

  int go() throws IOException {
    try {
      return doRun();
    } catch (FlagUsageException e) {
      System.err.println(e.getMessage());
      System.exit(1);
      return 1;
    }
  }

  @Override
  protected Compiler createCompiler() {
    return compiler;
  }

  @Override
  protected CompilerOptions createOptions() {
    CompilerOptions options = super.createOptions();
    options.setExportTestFunctions(exportTestFunctions);
    options.addWarningsGuard(warnings);
    options.setModuleResolutionMode(ModuleLoader.ResolutionMode.NODE);
    if (disablePropertyRenaming) {
        options.setPropertyRenaming(PropertyRenamingPolicy.OFF);
        options.setDisambiguateProperties(false);
    }
    if (devBuild) {
      applySpeedyOptions(options);
    }
    return options;
  }

  /**
   * Remove checks and undo most of the options set by the ADVANCED
   * CompilationLevel to make development builds faster.
   *
   * Does not change property renaming options so code that works when compiled
   * this way will also work with property renaming.
   *
   * @param options CompilerOptions object to update
   */
  private void applySpeedyOptions(CompilerOptions options) {
    options.resetWarningsGuard();

    options.setCheckSymbols(false);
    options.setCheckTypes(false);
    options.setInferTypes(false);
    options.setInferConst(false);
    options.setCheckSuspiciousCode(false);
    options.setRewritePolyfills(false);
    options.setIsolatePolyfills(false);
    options.setComputeFunctionSideEffects(false);

    // All the safe optimizations.
    options.setClosurePass(true); // Must be set to true for things to work
    options.setFoldConstants(false);
    options.setCoalesceVariableNames(false);
    options.setDeadAssignmentElimination(false);
    options.setExtractPrototypeMemberDeclarations(false);
    options.setCollapseVariableDeclarations(false);
    options.setConvertToDottedProperties(false);
    options.setLabelRenaming(false);
    options.setOptimizeArgumentsArray(false);
    options.setCollapseObjectLiterals(false);
    options.setProtectHiddenSideEffects(false);

    // All the advanced optimizations.
    options.setRemoveClosureAsserts(false);
    options.setRemoveAbstractMethods(false);
    options.setReserveRawExports(false);
    options.setRemoveUnusedPrototypeProperties(false);
    options.setRemoveUnusedClassProperties(false);
    options.setCollapseAnonymousFunctions(false);
    options.setWarningLevel(DiagnosticGroups.GLOBAL_THIS, CheckLevel.OFF);
    options.setRewriteFunctionExpressions(false);
    options.setSmartNameRemoval(false);
    options.setInlineConstantVars(false);
    options.setInlineFunctions(CompilerOptions.Reach.NONE);
    options.setAssumeClosuresOnlyCaptureReferences(false);
    options.setInlineVariables(CompilerOptions.Reach.NONE);
    options.setComputeFunctionSideEffects(false);
    options.setAssumeStrictThis(false);

    // Remove unused vars also removes unused functions.
    options.setRemoveUnusedVariables(CompilerOptions.Reach.NONE);

    // Move code around based on the defined modules.
    options.setCrossChunkCodeMotion(false);
    options.setCrossChunkMethodMotion(false);

    // Call optimizations
    options.setDevirtualizeMethods(false);
    options.setOptimizeCalls(false);

    // Type-based optimizations
    options.setAmbiguateProperties(false);
    options.setInlineProperties(false);
    options.setUseTypesForLocalOptimization(false);
  }
}
