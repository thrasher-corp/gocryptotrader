
	// Copyright 2020 ConsenSys Software Inc.
	//
	// Licensed under the Apache License, Version 2.0 (the "License");
	// you may not use this file except in compliance with the License.
	// You may obtain a copy of the License at
	//
	//     http://www.apache.org/licenses/LICENSE-2.0
	//
	// Unless required by applicable law or agreed to in writing, software
	// distributed under the License is distributed on an "AS IS" BASIS,
	// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	// See the License for the specific language governing permissions and
	// limitations under the License.
	
#include "textflag.h"
#include "funcdata.h"

TEXT ·mul(SB), NOSPLIT, $0-24

	// the algorithm is described here
	// https://hackmd.io/@zkteam/modular_multiplication
	// however, to benefit from the ADCX and ADOX carry chains
	// we split the inner loops in 2:
	// for i=0 to N-1
	// 		for j=0 to N-1
	// 		    (A,t[j])  := t[j] + x[j]*y[i] + A
	// 		m := t[0]*q'[0] mod W
	// 		C,_ := t[0] + m*q[0]
	// 		for j=1 to N-1
	// 		    (C,t[j-1]) := t[j] + m*q[j] + C
	// 		t[N-1] = C + A
	
    CMPB ·supportAdx(SB), $0x0000000000000001
    JNE l5
    MOVQ x+8(FP), R14
    MOVQ y+16(FP), R15
    XORQ DX, DX
    MOVQ 0(R15), DX
    MULXQ 0(R14), CX, BX
    MULXQ 8(R14), AX, BP
    ADOXQ AX, BX
    MULXQ 16(R14), AX, SI
    ADOXQ AX, BP
    MULXQ 24(R14), AX, DI
    ADOXQ AX, SI
    // add the last carries to DI
    MOVQ $0x0000000000000000, DX
    ADCXQ DX, DI
    ADOXQ DX, DI
    MOVQ CX, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    // C,_ := t[0] + m*q[0]
    MULXQ ·qElement+0(SB), AX, R8
    ADCXQ CX, AX
    MOVQ R8, CX
    // for j=1 to N-1
    //     (C,t[j-1]) := t[j] + m*q[j] + C
    ADCXQ BX, CX
    MULXQ ·qElement+8(SB), AX, BX
    ADOXQ AX, CX
    ADCXQ BP, BX
    MULXQ ·qElement+16(SB), AX, BP
    ADOXQ AX, BX
    ADCXQ SI, BP
    MULXQ ·qElement+24(SB), AX, SI
    ADOXQ AX, BP
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, SI
    ADOXQ DI, SI
    XORQ DX, DX
    MOVQ 8(R15), DX
    MULXQ 0(R14), AX, DI
    ADOXQ AX, CX
    ADCXQ DI, BX
    MULXQ 8(R14), AX, DI
    ADOXQ AX, BX
    ADCXQ DI, BP
    MULXQ 16(R14), AX, DI
    ADOXQ AX, BP
    ADCXQ DI, SI
    MULXQ 24(R14), AX, DI
    ADOXQ AX, SI
    // add the last carries to DI
    MOVQ $0x0000000000000000, DX
    ADCXQ DX, DI
    ADOXQ DX, DI
    MOVQ CX, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    // C,_ := t[0] + m*q[0]
    MULXQ ·qElement+0(SB), AX, R9
    ADCXQ CX, AX
    MOVQ R9, CX
    // for j=1 to N-1
    //     (C,t[j-1]) := t[j] + m*q[j] + C
    ADCXQ BX, CX
    MULXQ ·qElement+8(SB), AX, BX
    ADOXQ AX, CX
    ADCXQ BP, BX
    MULXQ ·qElement+16(SB), AX, BP
    ADOXQ AX, BX
    ADCXQ SI, BP
    MULXQ ·qElement+24(SB), AX, SI
    ADOXQ AX, BP
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, SI
    ADOXQ DI, SI
    XORQ DX, DX
    MOVQ 16(R15), DX
    MULXQ 0(R14), AX, DI
    ADOXQ AX, CX
    ADCXQ DI, BX
    MULXQ 8(R14), AX, DI
    ADOXQ AX, BX
    ADCXQ DI, BP
    MULXQ 16(R14), AX, DI
    ADOXQ AX, BP
    ADCXQ DI, SI
    MULXQ 24(R14), AX, DI
    ADOXQ AX, SI
    // add the last carries to DI
    MOVQ $0x0000000000000000, DX
    ADCXQ DX, DI
    ADOXQ DX, DI
    MOVQ CX, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    // C,_ := t[0] + m*q[0]
    MULXQ ·qElement+0(SB), AX, R10
    ADCXQ CX, AX
    MOVQ R10, CX
    // for j=1 to N-1
    //     (C,t[j-1]) := t[j] + m*q[j] + C
    ADCXQ BX, CX
    MULXQ ·qElement+8(SB), AX, BX
    ADOXQ AX, CX
    ADCXQ BP, BX
    MULXQ ·qElement+16(SB), AX, BP
    ADOXQ AX, BX
    ADCXQ SI, BP
    MULXQ ·qElement+24(SB), AX, SI
    ADOXQ AX, BP
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, SI
    ADOXQ DI, SI
    XORQ DX, DX
    MOVQ 24(R15), DX
    MULXQ 0(R14), AX, DI
    ADOXQ AX, CX
    ADCXQ DI, BX
    MULXQ 8(R14), AX, DI
    ADOXQ AX, BX
    ADCXQ DI, BP
    MULXQ 16(R14), AX, DI
    ADOXQ AX, BP
    ADCXQ DI, SI
    MULXQ 24(R14), AX, DI
    ADOXQ AX, SI
    // add the last carries to DI
    MOVQ $0x0000000000000000, DX
    ADCXQ DX, DI
    ADOXQ DX, DI
    MOVQ CX, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    // C,_ := t[0] + m*q[0]
    MULXQ ·qElement+0(SB), AX, R11
    ADCXQ CX, AX
    MOVQ R11, CX
    // for j=1 to N-1
    //     (C,t[j-1]) := t[j] + m*q[j] + C
    ADCXQ BX, CX
    MULXQ ·qElement+8(SB), AX, BX
    ADOXQ AX, CX
    ADCXQ BP, BX
    MULXQ ·qElement+16(SB), AX, BP
    ADOXQ AX, BX
    ADCXQ SI, BP
    MULXQ ·qElement+24(SB), AX, SI
    ADOXQ AX, BP
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, SI
    ADOXQ DI, SI
    MOVQ res+0(FP), R12
    MOVQ CX, R13
    MOVQ BX, R8
    MOVQ BP, R9
    MOVQ SI, R10
    SUBQ ·qElement+0(SB), R13
    SBBQ ·qElement+8(SB), R8
    SBBQ ·qElement+16(SB), R9
    SBBQ ·qElement+24(SB), R10
    CMOVQCC R13, CX
    CMOVQCC R8, BX
    CMOVQCC R9, BP
    CMOVQCC R10, SI
    MOVQ CX, 0(R12)
    MOVQ BX, 8(R12)
    MOVQ BP, 16(R12)
    MOVQ SI, 24(R12)
    RET
l5:
    MOVQ x+8(FP), R15
    MOVQ y+16(FP), R14
    MOVQ 0(R15), AX
    MOVQ 0(R14), R8
    MULQ R8
    MOVQ AX, CX
    MOVQ DX, R9
    MOVQ ·qElementInv0(SB), R10
    IMULQ CX, R10
    MOVQ $0x3c208c16d87cfd47, AX
    MULQ R10
    ADDQ CX, AX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, DI
    MOVQ 8(R15), AX
    MULQ R8
    MOVQ R9, BX
    ADDQ AX, BX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x97816a916871ca8d, AX
    MULQ R10
    ADDQ BX, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, CX
    MOVQ DX, DI
    MOVQ 16(R15), AX
    MULQ R8
    MOVQ R9, BP
    ADDQ AX, BP
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0xb85045b68181585d, AX
    MULQ R10
    ADDQ BP, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BX
    MOVQ DX, DI
    MOVQ 24(R15), AX
    MULQ R8
    MOVQ R9, SI
    ADDQ AX, SI
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x30644e72e131a029, AX
    MULQ R10
    ADDQ SI, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BP
    MOVQ DX, DI
    ADDQ DI, R9
    MOVQ R9, SI
    MOVQ 0(R15), AX
    MOVQ 8(R14), R8
    MULQ R8
    ADDQ AX, CX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ ·qElementInv0(SB), R10
    IMULQ CX, R10
    MOVQ $0x3c208c16d87cfd47, AX
    MULQ R10
    ADDQ CX, AX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, DI
    MOVQ 8(R15), AX
    MULQ R8
    ADDQ R9, BX
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x97816a916871ca8d, AX
    MULQ R10
    ADDQ BX, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, CX
    MOVQ DX, DI
    MOVQ 16(R15), AX
    MULQ R8
    ADDQ R9, BP
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BP
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0xb85045b68181585d, AX
    MULQ R10
    ADDQ BP, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BX
    MOVQ DX, DI
    MOVQ 24(R15), AX
    MULQ R8
    ADDQ R9, SI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, SI
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x30644e72e131a029, AX
    MULQ R10
    ADDQ SI, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BP
    MOVQ DX, DI
    ADDQ DI, R9
    MOVQ R9, SI
    MOVQ 0(R15), AX
    MOVQ 16(R14), R8
    MULQ R8
    ADDQ AX, CX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ ·qElementInv0(SB), R10
    IMULQ CX, R10
    MOVQ $0x3c208c16d87cfd47, AX
    MULQ R10
    ADDQ CX, AX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, DI
    MOVQ 8(R15), AX
    MULQ R8
    ADDQ R9, BX
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x97816a916871ca8d, AX
    MULQ R10
    ADDQ BX, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, CX
    MOVQ DX, DI
    MOVQ 16(R15), AX
    MULQ R8
    ADDQ R9, BP
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BP
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0xb85045b68181585d, AX
    MULQ R10
    ADDQ BP, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BX
    MOVQ DX, DI
    MOVQ 24(R15), AX
    MULQ R8
    ADDQ R9, SI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, SI
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x30644e72e131a029, AX
    MULQ R10
    ADDQ SI, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BP
    MOVQ DX, DI
    ADDQ DI, R9
    MOVQ R9, SI
    MOVQ 0(R15), AX
    MOVQ 24(R14), R8
    MULQ R8
    ADDQ AX, CX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ ·qElementInv0(SB), R10
    IMULQ CX, R10
    MOVQ $0x3c208c16d87cfd47, AX
    MULQ R10
    ADDQ CX, AX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, DI
    MOVQ 8(R15), AX
    MULQ R8
    ADDQ R9, BX
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x97816a916871ca8d, AX
    MULQ R10
    ADDQ BX, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, CX
    MOVQ DX, DI
    MOVQ 16(R15), AX
    MULQ R8
    ADDQ R9, BP
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BP
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0xb85045b68181585d, AX
    MULQ R10
    ADDQ BP, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BX
    MOVQ DX, DI
    MOVQ 24(R15), AX
    MULQ R8
    ADDQ R9, SI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, SI
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x30644e72e131a029, AX
    MULQ R10
    ADDQ SI, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BP
    MOVQ DX, DI
    ADDQ DI, R9
    MOVQ R9, SI
    MOVQ res+0(FP), R15
    MOVQ CX, R11
    MOVQ BX, R12
    MOVQ BP, R13
    MOVQ SI, DI
    SUBQ ·qElement+0(SB), R11
    SBBQ ·qElement+8(SB), R12
    SBBQ ·qElement+16(SB), R13
    SBBQ ·qElement+24(SB), DI
    CMOVQCC R11, CX
    CMOVQCC R12, BX
    CMOVQCC R13, BP
    CMOVQCC DI, SI
    MOVQ CX, 0(R15)
    MOVQ BX, 8(R15)
    MOVQ BP, 16(R15)
    MOVQ SI, 24(R15)
    RET

TEXT ·square(SB), NOSPLIT, $0-16

	// the algorithm is described here
	// https://hackmd.io/@zkteam/modular_multiplication
	// for i=0 to N-1
	// A, t[i] = x[i] * x[i] + t[i]
	// p = 0
	// for j=i+1 to N-1
	//     p,A,t[j] = 2*x[j]*x[i] + t[j] + (p,A)
	// m = t[0] * q'[0]
	// C, _ = t[0] + q[0]*m
	// for j=1 to N-1
	//     C, t[j-1] = q[j]*m +  t[j] + C
	// t[N-1] = C + A

	
    CMPB ·supportAdx(SB), $0x0000000000000001
    JNE l6
    MOVQ x+8(FP), BP
    XORQ AX, AX
    MOVQ 0(BP), DX
    MULXQ 8(BP), DI, R8
    MULXQ 16(BP), AX, R9
    ADCXQ AX, R8
    MULXQ 24(BP), AX, SI
    ADCXQ AX, R9
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, SI
    XORQ AX, AX
    MULXQ DX, R14, DX
    ADCXQ DI, DI
    MOVQ DI, R15
    ADOXQ DX, R15
    ADCXQ R8, R8
    MOVQ R8, CX
    ADOXQ AX, CX
    ADCXQ R9, R9
    MOVQ R9, BX
    ADOXQ AX, BX
    ADCXQ SI, SI
    ADOXQ AX, SI
    MOVQ R14, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    MULXQ ·qElement+0(SB), AX, R10
    ADCXQ R14, AX
    MOVQ R10, R14
    ADCXQ R15, R14
    MULXQ ·qElement+8(SB), AX, R15
    ADOXQ AX, R14
    ADCXQ CX, R15
    MULXQ ·qElement+16(SB), AX, CX
    ADOXQ AX, R15
    ADCXQ BX, CX
    MULXQ ·qElement+24(SB), AX, BX
    ADOXQ AX, CX
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, BX
    ADOXQ SI, BX
    XORQ AX, AX
    MOVQ 8(BP), DX
    MULXQ 16(BP), R11, R12
    MULXQ 24(BP), AX, SI
    ADCXQ AX, R12
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, SI
    XORQ AX, AX
    ADCXQ R11, R11
    ADOXQ R11, CX
    ADCXQ R12, R12
    ADOXQ R12, BX
    ADCXQ SI, SI
    ADOXQ AX, SI
    XORQ AX, AX
    MULXQ DX, AX, DX
    ADOXQ AX, R15
    MOVQ $0x0000000000000000, AX
    ADOXQ DX, CX
    ADOXQ AX, BX
    ADOXQ AX, SI
    MOVQ R14, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    MULXQ ·qElement+0(SB), AX, R13
    ADCXQ R14, AX
    MOVQ R13, R14
    ADCXQ R15, R14
    MULXQ ·qElement+8(SB), AX, R15
    ADOXQ AX, R14
    ADCXQ CX, R15
    MULXQ ·qElement+16(SB), AX, CX
    ADOXQ AX, R15
    ADCXQ BX, CX
    MULXQ ·qElement+24(SB), AX, BX
    ADOXQ AX, CX
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, BX
    ADOXQ SI, BX
    XORQ AX, AX
    MOVQ 16(BP), DX
    MULXQ 24(BP), DI, SI
    ADCXQ DI, DI
    ADOXQ DI, BX
    ADCXQ SI, SI
    ADOXQ AX, SI
    XORQ AX, AX
    MULXQ DX, AX, DX
    ADOXQ AX, CX
    MOVQ $0x0000000000000000, AX
    ADOXQ DX, BX
    ADOXQ AX, SI
    MOVQ R14, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    MULXQ ·qElement+0(SB), AX, R8
    ADCXQ R14, AX
    MOVQ R8, R14
    ADCXQ R15, R14
    MULXQ ·qElement+8(SB), AX, R15
    ADOXQ AX, R14
    ADCXQ CX, R15
    MULXQ ·qElement+16(SB), AX, CX
    ADOXQ AX, R15
    ADCXQ BX, CX
    MULXQ ·qElement+24(SB), AX, BX
    ADOXQ AX, CX
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, BX
    ADOXQ SI, BX
    XORQ AX, AX
    MOVQ 24(BP), DX
    MULXQ DX, AX, SI
    ADCXQ AX, BX
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, SI
    MOVQ R14, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    MULXQ ·qElement+0(SB), AX, R9
    ADCXQ R14, AX
    MOVQ R9, R14
    ADCXQ R15, R14
    MULXQ ·qElement+8(SB), AX, R15
    ADOXQ AX, R14
    ADCXQ CX, R15
    MULXQ ·qElement+16(SB), AX, CX
    ADOXQ AX, R15
    ADCXQ BX, CX
    MULXQ ·qElement+24(SB), AX, BX
    ADOXQ AX, CX
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, BX
    ADOXQ SI, BX
    MOVQ res+0(FP), R10
    MOVQ R14, R11
    MOVQ R15, R12
    MOVQ CX, R13
    MOVQ BX, DI
    SUBQ ·qElement+0(SB), R11
    SBBQ ·qElement+8(SB), R12
    SBBQ ·qElement+16(SB), R13
    SBBQ ·qElement+24(SB), DI
    CMOVQCC R11, R14
    CMOVQCC R12, R15
    CMOVQCC R13, CX
    CMOVQCC DI, BX
    MOVQ R14, 0(R10)
    MOVQ R15, 8(R10)
    MOVQ CX, 16(R10)
    MOVQ BX, 24(R10)
    RET
l6:
    MOVQ x+8(FP), R15
    MOVQ x+8(FP), R14
    MOVQ 0(R15), AX
    MOVQ 0(R14), R8
    MULQ R8
    MOVQ AX, CX
    MOVQ DX, R9
    MOVQ ·qElementInv0(SB), R10
    IMULQ CX, R10
    MOVQ $0x3c208c16d87cfd47, AX
    MULQ R10
    ADDQ CX, AX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, DI
    MOVQ 8(R15), AX
    MULQ R8
    MOVQ R9, BX
    ADDQ AX, BX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x97816a916871ca8d, AX
    MULQ R10
    ADDQ BX, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, CX
    MOVQ DX, DI
    MOVQ 16(R15), AX
    MULQ R8
    MOVQ R9, BP
    ADDQ AX, BP
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0xb85045b68181585d, AX
    MULQ R10
    ADDQ BP, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BX
    MOVQ DX, DI
    MOVQ 24(R15), AX
    MULQ R8
    MOVQ R9, SI
    ADDQ AX, SI
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x30644e72e131a029, AX
    MULQ R10
    ADDQ SI, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BP
    MOVQ DX, DI
    ADDQ DI, R9
    MOVQ R9, SI
    MOVQ 0(R15), AX
    MOVQ 8(R14), R8
    MULQ R8
    ADDQ AX, CX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ ·qElementInv0(SB), R10
    IMULQ CX, R10
    MOVQ $0x3c208c16d87cfd47, AX
    MULQ R10
    ADDQ CX, AX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, DI
    MOVQ 8(R15), AX
    MULQ R8
    ADDQ R9, BX
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x97816a916871ca8d, AX
    MULQ R10
    ADDQ BX, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, CX
    MOVQ DX, DI
    MOVQ 16(R15), AX
    MULQ R8
    ADDQ R9, BP
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BP
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0xb85045b68181585d, AX
    MULQ R10
    ADDQ BP, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BX
    MOVQ DX, DI
    MOVQ 24(R15), AX
    MULQ R8
    ADDQ R9, SI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, SI
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x30644e72e131a029, AX
    MULQ R10
    ADDQ SI, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BP
    MOVQ DX, DI
    ADDQ DI, R9
    MOVQ R9, SI
    MOVQ 0(R15), AX
    MOVQ 16(R14), R8
    MULQ R8
    ADDQ AX, CX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ ·qElementInv0(SB), R10
    IMULQ CX, R10
    MOVQ $0x3c208c16d87cfd47, AX
    MULQ R10
    ADDQ CX, AX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, DI
    MOVQ 8(R15), AX
    MULQ R8
    ADDQ R9, BX
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x97816a916871ca8d, AX
    MULQ R10
    ADDQ BX, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, CX
    MOVQ DX, DI
    MOVQ 16(R15), AX
    MULQ R8
    ADDQ R9, BP
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BP
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0xb85045b68181585d, AX
    MULQ R10
    ADDQ BP, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BX
    MOVQ DX, DI
    MOVQ 24(R15), AX
    MULQ R8
    ADDQ R9, SI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, SI
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x30644e72e131a029, AX
    MULQ R10
    ADDQ SI, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BP
    MOVQ DX, DI
    ADDQ DI, R9
    MOVQ R9, SI
    MOVQ 0(R15), AX
    MOVQ 24(R14), R8
    MULQ R8
    ADDQ AX, CX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ ·qElementInv0(SB), R10
    IMULQ CX, R10
    MOVQ $0x3c208c16d87cfd47, AX
    MULQ R10
    ADDQ CX, AX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, DI
    MOVQ 8(R15), AX
    MULQ R8
    ADDQ R9, BX
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BX
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x97816a916871ca8d, AX
    MULQ R10
    ADDQ BX, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, CX
    MOVQ DX, DI
    MOVQ 16(R15), AX
    MULQ R8
    ADDQ R9, BP
    ADCQ $0x0000000000000000, DX
    ADDQ AX, BP
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0xb85045b68181585d, AX
    MULQ R10
    ADDQ BP, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BX
    MOVQ DX, DI
    MOVQ 24(R15), AX
    MULQ R8
    ADDQ R9, SI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, SI
    ADCQ $0x0000000000000000, DX
    MOVQ DX, R9
    MOVQ $0x30644e72e131a029, AX
    MULQ R10
    ADDQ SI, DI
    ADCQ $0x0000000000000000, DX
    ADDQ AX, DI
    ADCQ $0x0000000000000000, DX
    MOVQ DI, BP
    MOVQ DX, DI
    ADDQ DI, R9
    MOVQ R9, SI
    MOVQ res+0(FP), R15
    MOVQ CX, R11
    MOVQ BX, R12
    MOVQ BP, R13
    MOVQ SI, DI
    SUBQ ·qElement+0(SB), R11
    SBBQ ·qElement+8(SB), R12
    SBBQ ·qElement+16(SB), R13
    SBBQ ·qElement+24(SB), DI
    CMOVQCC R11, CX
    CMOVQCC R12, BX
    CMOVQCC R13, BP
    CMOVQCC DI, SI
    MOVQ CX, 0(R15)
    MOVQ BX, 8(R15)
    MOVQ BP, 16(R15)
    MOVQ SI, 24(R15)
    RET

TEXT ·fromMont(SB), $8-8
NO_LOCAL_POINTERS

	// the algorithm is described here
	// https://hackmd.io/@zkteam/modular_multiplication
	// when y = 1 we have: 
	// for i=0 to N-1
	// 		t[i] = x[i]
	// for i=0 to N-1
	// 		m := t[0]*q'[0] mod W
	// 		C,_ := t[0] + m*q[0]
	// 		for j=1 to N-1
	// 		    (C,t[j-1]) := t[j] + m*q[j] + C
	// 		t[N-1] = C
    CMPB ·supportAdx(SB), $0x0000000000000001
    JNE l7
    MOVQ res+0(FP), BP
    MOVQ 0(BP), R14
    MOVQ 8(BP), R15
    MOVQ 16(BP), CX
    MOVQ 24(BP), BX
    XORQ DX, DX
    MOVQ R14, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    // C,_ := t[0] + m*q[0]
    MULXQ ·qElement+0(SB), AX, SI
    ADCXQ R14, AX
    MOVQ SI, R14
    // for j=1 to N-1
    //     (C,t[j-1]) := t[j] + m*q[j] + C
    ADCXQ R15, R14
    MULXQ ·qElement+8(SB), AX, R15
    ADOXQ AX, R14
    ADCXQ CX, R15
    MULXQ ·qElement+16(SB), AX, CX
    ADOXQ AX, R15
    ADCXQ BX, CX
    MULXQ ·qElement+24(SB), AX, BX
    ADOXQ AX, CX
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, BX
    ADOXQ AX, BX
    XORQ DX, DX
    MOVQ R14, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    // C,_ := t[0] + m*q[0]
    MULXQ ·qElement+0(SB), AX, SI
    ADCXQ R14, AX
    MOVQ SI, R14
    // for j=1 to N-1
    //     (C,t[j-1]) := t[j] + m*q[j] + C
    ADCXQ R15, R14
    MULXQ ·qElement+8(SB), AX, R15
    ADOXQ AX, R14
    ADCXQ CX, R15
    MULXQ ·qElement+16(SB), AX, CX
    ADOXQ AX, R15
    ADCXQ BX, CX
    MULXQ ·qElement+24(SB), AX, BX
    ADOXQ AX, CX
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, BX
    ADOXQ AX, BX
    XORQ DX, DX
    MOVQ R14, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    // C,_ := t[0] + m*q[0]
    MULXQ ·qElement+0(SB), AX, SI
    ADCXQ R14, AX
    MOVQ SI, R14
    // for j=1 to N-1
    //     (C,t[j-1]) := t[j] + m*q[j] + C
    ADCXQ R15, R14
    MULXQ ·qElement+8(SB), AX, R15
    ADOXQ AX, R14
    ADCXQ CX, R15
    MULXQ ·qElement+16(SB), AX, CX
    ADOXQ AX, R15
    ADCXQ BX, CX
    MULXQ ·qElement+24(SB), AX, BX
    ADOXQ AX, CX
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, BX
    ADOXQ AX, BX
    XORQ DX, DX
    MOVQ R14, DX
    MULXQ ·qElementInv0(SB), DX, AX                        // m := t[0]*q'[0] mod W
    XORQ AX, AX
    // C,_ := t[0] + m*q[0]
    MULXQ ·qElement+0(SB), AX, SI
    ADCXQ R14, AX
    MOVQ SI, R14
    // for j=1 to N-1
    //     (C,t[j-1]) := t[j] + m*q[j] + C
    ADCXQ R15, R14
    MULXQ ·qElement+8(SB), AX, R15
    ADOXQ AX, R14
    ADCXQ CX, R15
    MULXQ ·qElement+16(SB), AX, CX
    ADOXQ AX, R15
    ADCXQ BX, CX
    MULXQ ·qElement+24(SB), AX, BX
    ADOXQ AX, CX
    MOVQ $0x0000000000000000, AX
    ADCXQ AX, BX
    ADOXQ AX, BX
    MOVQ R14, DI
    MOVQ R15, R8
    MOVQ CX, R9
    MOVQ BX, R10
    SUBQ ·qElement+0(SB), DI
    SBBQ ·qElement+8(SB), R8
    SBBQ ·qElement+16(SB), R9
    SBBQ ·qElement+24(SB), R10
    CMOVQCC DI, R14
    CMOVQCC R8, R15
    CMOVQCC R9, CX
    CMOVQCC R10, BX
    MOVQ R14, 0(BP)
    MOVQ R15, 8(BP)
    MOVQ CX, 16(BP)
    MOVQ BX, 24(BP)
    RET
l7:
    MOVQ res+0(FP), AX
    MOVQ AX, (SP)
CALL ·_fromMontGeneric(SB)
    RET

TEXT ·reduce(SB), NOSPLIT, $0-8
    MOVQ res+0(FP), AX
    MOVQ 0(AX), DX
    MOVQ 8(AX), CX
    MOVQ 16(AX), BX
    MOVQ 24(AX), BP
    MOVQ DX, SI
    MOVQ CX, DI
    MOVQ BX, R8
    MOVQ BP, R9
    SUBQ ·qElement+0(SB), SI
    SBBQ ·qElement+8(SB), DI
    SBBQ ·qElement+16(SB), R8
    SBBQ ·qElement+24(SB), R9
    CMOVQCC SI, DX
    CMOVQCC DI, CX
    CMOVQCC R8, BX
    CMOVQCC R9, BP
    MOVQ DX, 0(AX)
    MOVQ CX, 8(AX)
    MOVQ BX, 16(AX)
    MOVQ BP, 24(AX)
    RET

TEXT ·add(SB), NOSPLIT, $0-24
    MOVQ x+8(FP), AX
    MOVQ 0(AX), BX
    MOVQ 8(AX), BP
    MOVQ 16(AX), SI
    MOVQ 24(AX), DI
    MOVQ y+16(FP), DX
    ADDQ 0(DX), BX
    ADCQ 8(DX), BP
    ADCQ 16(DX), SI
    ADCQ 24(DX), DI
    MOVQ res+0(FP), CX
    MOVQ BX, R8
    MOVQ BP, R9
    MOVQ SI, R10
    MOVQ DI, R11
    SUBQ ·qElement+0(SB), R8
    SBBQ ·qElement+8(SB), R9
    SBBQ ·qElement+16(SB), R10
    SBBQ ·qElement+24(SB), R11
    CMOVQCC R8, BX
    CMOVQCC R9, BP
    CMOVQCC R10, SI
    CMOVQCC R11, DI
    MOVQ BX, 0(CX)
    MOVQ BP, 8(CX)
    MOVQ SI, 16(CX)
    MOVQ DI, 24(CX)
    RET

TEXT ·sub(SB), NOSPLIT, $0-24
    MOVQ x+8(FP), BP
    MOVQ 0(BP), AX
    MOVQ 8(BP), DX
    MOVQ 16(BP), CX
    MOVQ 24(BP), BX
    MOVQ y+16(FP), SI
    SUBQ 0(SI), AX
    SBBQ 8(SI), DX
    SBBQ 16(SI), CX
    SBBQ 24(SI), BX
    MOVQ $0x3c208c16d87cfd47, DI
    MOVQ $0x97816a916871ca8d, R8
    MOVQ $0xb85045b68181585d, R9
    MOVQ $0x30644e72e131a029, R10
    MOVQ $0x0000000000000000, R11
    CMOVQCC R11, DI
    CMOVQCC R11, R8
    CMOVQCC R11, R9
    CMOVQCC R11, R10
    ADDQ DI, AX
    ADCQ R8, DX
    ADCQ R9, CX
    ADCQ R10, BX
    MOVQ res+0(FP), R12
    MOVQ AX, 0(R12)
    MOVQ DX, 8(R12)
    MOVQ CX, 16(R12)
    MOVQ BX, 24(R12)
    RET

TEXT ·double(SB), NOSPLIT, $0-16
    MOVQ res+0(FP), DX
    MOVQ x+8(FP), AX
    MOVQ 0(AX), CX
    MOVQ 8(AX), BX
    MOVQ 16(AX), BP
    MOVQ 24(AX), SI
    ADDQ CX, CX
    ADCQ BX, BX
    ADCQ BP, BP
    ADCQ SI, SI
    MOVQ CX, DI
    MOVQ BX, R8
    MOVQ BP, R9
    MOVQ SI, R10
    SUBQ ·qElement+0(SB), DI
    SBBQ ·qElement+8(SB), R8
    SBBQ ·qElement+16(SB), R9
    SBBQ ·qElement+24(SB), R10
    CMOVQCC DI, CX
    CMOVQCC R8, BX
    CMOVQCC R9, BP
    CMOVQCC R10, SI
    MOVQ CX, 0(DX)
    MOVQ BX, 8(DX)
    MOVQ BP, 16(DX)
    MOVQ SI, 24(DX)
    RET

TEXT ·neg(SB), NOSPLIT, $0-16
    MOVQ res+0(FP), DX
    MOVQ x+8(FP), AX
    MOVQ 0(AX), BX
    MOVQ 8(AX), BP
    MOVQ 16(AX), SI
    MOVQ 24(AX), DI
    MOVQ BX, AX
    ORQ BP, AX
    ORQ SI, AX
    ORQ DI, AX
    TESTQ AX, AX
    JNE l8
    MOVQ AX, 0(DX)
    MOVQ AX, 8(DX)
    RET
l8:
    MOVQ $0x3c208c16d87cfd47, CX
    SUBQ BX, CX
    MOVQ CX, 0(DX)
    MOVQ $0x97816a916871ca8d, CX
    SBBQ BP, CX
    MOVQ CX, 8(DX)
    MOVQ $0xb85045b68181585d, CX
    SBBQ SI, CX
    MOVQ CX, 16(DX)
    MOVQ $0x30644e72e131a029, CX
    SBBQ DI, CX
    MOVQ CX, 24(DX)
    RET
