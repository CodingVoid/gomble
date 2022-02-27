#include "CryptState.h"
#include <stdio.h>

unsigned char msg[] = {
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
};

unsigned char rawkey[] = { 0x96, 0x8b, 0x1b, 0x0c, 0x53, 0x1e, 0x1f, 0x80, 0xa6, 0x1d, 0xcb, 0x27, 0x94, 0x09, 0x6f, 0x32, };
unsigned char encrypt_iv[] = { 0x1e, 0x2a, 0x9b, 0xd0, 0x2d, 0xa6, 0x8e, 0x46, 0x26, 0x85, 0x83, 0xe9, 0x14, 0x2a, 0xff, 0x2a, };
unsigned char decrypt_iv[] = { 0x73, 0x99, 0x9d, 0xa2, 0x03, 0x70, 0x00, 0x96, 0xef, 0x55, 0x06, 0x7a, 0x8b, 0xbe, 0x00, 0x07, };
unsigned char encrypted[] = { 0x1f, 0xfc, 0xdd, 0xb4, 0x68, 0x13, 0x68, 0xb7, 0x92, 0x67, 0xca, 0x2d, 0xba, 0xb7, 0x0d, 0x44, 0xdf, 0x32, 0xd4, };


static void DumpBytes(unsigned char *bytes, unsigned int len, const char *name) {
	printf("unsigned char %s[] = { ", name);
	for (int i = 0; i < len; i++) {
		printf("0x%.2x, ", bytes[i]);
	}
	printf("}\n");
}

int main(int argc, char *argv[]) {
	MumbleClient::CryptState cs;
//	cs.genKey();
	cs.setKey(rawkey, encrypt_iv, decrypt_iv);

	DumpBytes(cs.raw_key, AES_BLOCK_SIZE, "rawkey");
	DumpBytes(cs.encrypt_iv, AES_BLOCK_SIZE, "encrypt_iv");
	DumpBytes(cs.decrypt_iv, AES_BLOCK_SIZE, "decrypt_iv");

	unsigned char buf[19];
	cs.encrypt(msg, &buf[0], 15);

	DumpBytes(buf, 19, "encrypted");
	DumpBytes(cs.encrypt_iv, AES_BLOCK_SIZE, "post_eiv");
}
