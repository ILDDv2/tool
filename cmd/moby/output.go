package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/linuxkit/linuxkit/src/initrd"
)

const (
	bios = "linuxkit/aarch64/mkimage-iso-bios:afc9d3470557101f53aed9784b5215f8cc05a029"
	efi  = "linuxkit/aarch64/mkimage-iso-efi:29204397d5128dbe6df31d0187fd706239b0f862"
	gcp  = "linuxkit/mkimage-gcp:46716b3d3f7aa1a7607a3426fe0ccebc554b14ee@sha256:18d8e0482f65a2481f5b6ba1e7ce77723b246bf13bdb612be5e64df90297940c"
	img  = "linuxkit/aarch64/mkimage-img-gz:dcd6839dc5ee1c67e5ddb2de308ed8a355f4bc5d"
	qcow = "linuxkit/mkimage-qcow:69890f35b55e4ff8a2c7a714907f988e57056d02@sha256:f89dc09f82bdbf86d7edae89604544f20b99d99c9b5cabcf1f93308095d8c244"
	vhd  = "linuxkit/mkimage-vhd:a04c8480d41ca9cef6b7710bd45a592220c3acb2@sha256:ba373dc8ae5dc72685dbe4b872d8f588bc68b2114abd8bdc6a74d82a2b62cce3"
	vmdk = "linuxkit/mkimage-vmdk:182b541474ca7965c8e8f987389b651859f760da@sha256:99638c5ddb17614f54c6b8e11bd9d49d1dea9d837f38e0f6c1a5f451085d449b"
)

func outputs(m *Moby, base string, image []byte) error {
	log.Debugf("output: %s %s", m.Outputs, base)

	for _, o := range m.Outputs {
		switch o.Format {
		case "tar":
			err := outputTar(base, image)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "kernel+initrd":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputKernelInitrd(base, kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "iso-bios":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputImg(bios, base+".iso", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "iso-efi":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputImg(efi, base+"-efi.iso", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "img-gz":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputImgSize(img, base+".img.gz", kernel, initrd, cmdline, "1G")
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "gcp-img":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputImg(gcp, base+".img.tar.gz", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "qcow", "qcow2":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputImg(qcow, base+".qcow2", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "vhd":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputImg(vhd, base+".vhd", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "vmdk":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputImg(vmdk, base+".vmdk", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "":
			return fmt.Errorf("No format specified for output")
		default:
			return fmt.Errorf("Unknown output type %s", o.Format)
		}
	}
	return nil
}

func tarToInitrd(image []byte) ([]byte, []byte, string, error) {
	w := new(bytes.Buffer)
	iw := initrd.NewWriter(w)
	r := bytes.NewReader(image)
	tr := tar.NewReader(r)
	kernel, cmdline, err := initrd.CopySplitTar(iw, tr)
	if err != nil {
		return []byte{}, []byte{}, "", err
	}
	iw.Close()
	return kernel, w.Bytes(), cmdline, nil
}

func tarInitrdKernel(kernel, initrd []byte, cmdline string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	hdr := &tar.Header{
		Name: "kernel",
		Mode: 0600,
		Size: int64(len(kernel)),
	}
	err := tw.WriteHeader(hdr)
	if err != nil {
		return buf, err
	}
	_, err = tw.Write(kernel)
	if err != nil {
		return buf, err
	}
	hdr = &tar.Header{
		Name: "initrd.img",
		Mode: 0600,
		Size: int64(len(initrd)),
	}
	err = tw.WriteHeader(hdr)
	if err != nil {
		return buf, err
	}
	_, err = tw.Write(initrd)
	if err != nil {
		return buf, err
	}
	hdr = &tar.Header{
		Name: "cmdline",
		Mode: 0600,
		Size: int64(len(cmdline)),
	}
	err = tw.WriteHeader(hdr)
	if err != nil {
		return buf, err
	}
	_, err = tw.Write([]byte(cmdline))
	if err != nil {
		return buf, err
	}
	err = tw.Close()
	if err != nil {
		return buf, err
	}
	return buf, nil
}

func outputImg(image, filename string, kernel []byte, initrd []byte, cmdline string) error {
	log.Debugf("output img: %s %s", image, filename)
	log.Infof("  %s", filename)
	buf, err := tarInitrdKernel(kernel, initrd, cmdline)
	if err != nil {
		return err
	}
	img, err := dockerRunInput(buf, image, cmdline)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, img, os.FileMode(0644))
	if err != nil {
		return err
	}
	return nil
}

// this should replace the other version for types that can specify a size, and get size from CLI in future
func outputImgSize(image, filename string, kernel []byte, initrd []byte, cmdline string, size string) error {
	log.Debugf("output img: %s %s size %s", image, filename, size)
	log.Infof("  %s", filename)
	buf, err := tarInitrdKernel(kernel, initrd, cmdline)
	if err != nil {
		return err
	}
	var img []byte
	if size == "" {
		img, err = dockerRunInput(buf, image)
	} else {
		img, err = dockerRunInput(buf, image, size)
	}
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, img, os.FileMode(0644))
	if err != nil {
		return err
	}
	return nil
}

func outputKernelInitrd(base string, kernel []byte, initrd []byte, cmdline string) error {
	log.Debugf("output kernel/initrd: %s %s", base, cmdline)
	log.Infof("  %s %s %s", base+"-kernel", base+"-initrd.img", base+"-cmdline")
	err := ioutil.WriteFile(base+"-initrd.img", initrd, os.FileMode(0644))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(base+"-kernel", kernel, os.FileMode(0644))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(base+"-cmdline", []byte(cmdline), os.FileMode(0644))
	if err != nil {
		return err
	}
	return nil
}

func outputTar(base string, initrd []byte) error {
	log.Debugf("output tar: %s", base)
	log.Infof("  %s", base+".tar")
	return ioutil.WriteFile(base+".tar", initrd, os.FileMode(0644))
}
